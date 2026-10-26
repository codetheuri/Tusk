package middleware

import (
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"time"

	"github.com/codetheuri/tusk/v2/pkg/logger"
)

// DefaultMaxClients bounds how many callers a RateLimiter tracks at once.
// Each costs about a hundred bytes, so the default is around 10 MB at most.
const DefaultMaxClients = 100_000

// overflowKey is the one bucket every new caller shares while the table is
// full of callers still inside their limits. Strict on purpose: a flood of
// fresh addresses gets one allowance between them, not one each.
const overflowKey = "\x00overflow"

type clientVisitor struct {
	tokens   float64
	lastSeen time.Time
}

// RateLimiter is a token bucket per caller: maxTokens is how large a burst
// is allowed, refillRate how many requests per second are sustained.
//
// A caller is its IP address without the port (an IPv6 address by its /64,
// which one device can roam within freely). X-Forwarded-For is believed only
// from a trusted proxy (WithTrustedProxies); without one, the connection's
// own address is the caller. Earlier versions keyed on the address with its
// port, which every new connection changes, and on X-Forwarded-For as sent
// by anyone, which a client can set to anything: either way each request
// could arrive as a new caller with a full bucket.
type RateLimiter struct {
	mu         sync.Mutex
	visitors   map[string]*clientVisitor
	maxTokens  float64
	refillRate float64
	maxClients int
	trusted    []netip.Prefix
	log        logger.Logger
}

// RateLimitOption configures a RateLimiter.
type RateLimitOption func(*RateLimiter)

// WithTrustedProxies names the proxies whose X-Forwarded-For is believed.
// Only addresses that can reach the service directly belong here: a proxy
// in front of it, or the private network it is published on.
func WithTrustedProxies(prefixes []netip.Prefix) RateLimitOption {
	return func(rl *RateLimiter) { rl.trusted = prefixes }
}

// WithMaxClients changes DefaultMaxClients.
func WithMaxClients(n int) RateLimitOption {
	return func(rl *RateLimiter) {
		if n > 0 {
			rl.maxClients = n
		}
	}
}

// NewRateLimiter creates a RateLimiter with specified max tokens (capacity)
// and refill rate per second.
func NewRateLimiter(maxTokens float64, refillRate float64, log logger.Logger, opts ...RateLimitOption) *RateLimiter {
	rl := &RateLimiter{
		visitors:   make(map[string]*clientVisitor),
		maxTokens:  maxTokens,
		refillRate: refillRate,
		maxClients: DefaultMaxClients,
		log:        log,
	}
	for _, opt := range opts {
		opt(rl)
	}

	// Periodically clean up stale visitors (older than 10 minutes)
	go func() {
		for {
			time.Sleep(3 * time.Minute)
			rl.mu.Lock()
			for ip, v := range rl.visitors {
				if time.Since(v.lastSeen) > 10*time.Minute {
					delete(rl.visitors, ip)
				}
			}
			rl.mu.Unlock()
		}
	}()

	return rl
}

// Allow takes one request from key's bucket and reports whether there was
// one to take. The middleware calls it with the caller's address; a service
// can call it with anything else worth limiting, such as an account.
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	v, ok := rl.visitors[key]
	if !ok {
		if len(rl.visitors) >= rl.maxClients {
			rl.dropRefilled(now)
		}
		if len(rl.visitors) >= rl.maxClients {
			key = overflowKey
			v, ok = rl.visitors[key]
		}
		if !ok {
			v = &clientVisitor{tokens: rl.maxTokens, lastSeen: now}
			rl.visitors[key] = v
		}
	}

	v.tokens += now.Sub(v.lastSeen).Seconds() * rl.refillRate
	if v.tokens > rl.maxTokens {
		v.tokens = rl.maxTokens
	}
	v.lastSeen = now
	if v.tokens < 1 {
		return false
	}
	v.tokens--
	return true
}

// dropRefilled forgets callers whose bucket has filled up again: tracking
// them changes nothing, as a new caller starts full. Called with mu held.
func (rl *RateLimiter) dropRefilled(now time.Time) {
	for key, v := range rl.visitors {
		if key == overflowKey {
			continue
		}
		if v.tokens+now.Sub(v.lastSeen).Seconds()*rl.refillRate >= rl.maxTokens {
			delete(rl.visitors, key)
		}
	}
}

// Limit returns a middleware that limits incoming HTTP requests per caller.
func (rl *RateLimiter) Limit() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := ClientIP(r, rl.trusted)
			if !rl.Allow(callerKey(ip, r.RemoteAddr)) {
				rl.log.Warn("Rate limit exceeded", "ip", ip.String(), "path", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", "60")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"success":false,"message":"Too many requests. Please try again later."}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ClientIP is the address of whoever made r. When the connection comes from
// a trusted proxy, X-Forwarded-For is read from the right, past every
// trusted hop, to the first address none of them is: the one the nearest
// trusted proxy saw connect. Entries left of it are whatever the client
// sent, and are never believed. Returns the zero Addr if no address can be
// read at all.
func ClientIP(r *http.Request, trusted []netip.Prefix) netip.Addr {
	peer := parseAddr(r.RemoteAddr)
	if !peer.IsValid() || !isTrusted(peer, trusted) {
		return peer
	}
	hops := r.Header.Values("X-Forwarded-For")
	for i := len(hops) - 1; i >= 0; i-- {
		parts := strings.Split(hops[i], ",")
		for j := len(parts) - 1; j >= 0; j-- {
			addr := parseAddr(strings.TrimSpace(parts[j]))
			if !addr.IsValid() {
				// Garbage from a client, past which nothing can be trusted:
				// the last good hop is the best answer there is.
				return peer
			}
			if !isTrusted(addr, trusted) {
				return addr
			}
			peer = addr
		}
	}
	return peer
}

// ParseTrustedProxies reads a comma-separated list of addresses and CIDR
// ranges, such as "172.16.0.0/12, 127.0.0.1".
func ParseTrustedProxies(list string) ([]netip.Prefix, error) {
	var out []netip.Prefix
	for _, item := range strings.Split(list, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if strings.Contains(item, "/") {
			p, err := netip.ParsePrefix(item)
			if err != nil {
				return nil, fmt.Errorf("trusted proxy %q: %w", item, err)
			}
			out = append(out, p.Masked())
			continue
		}
		a, err := netip.ParseAddr(item)
		if err != nil {
			return nil, fmt.Errorf("trusted proxy %q: %w", item, err)
		}
		a = a.Unmap()
		out = append(out, netip.PrefixFrom(a, a.BitLen()))
	}
	return out, nil
}

// callerKey groups an IPv6 address by its /64. An unreadable address falls
// back to the raw RemoteAddr, port and all, rather than sharing one bucket
// with every other unreadable caller.
func callerKey(ip netip.Addr, remoteAddr string) string {
	if !ip.IsValid() {
		return remoteAddr
	}
	if ip.Is6() {
		p, _ := ip.Prefix(64)
		return p.String()
	}
	return ip.String()
}

// parseAddr reads "1.2.3.4", "1.2.3.4:5678", "::1" or "[::1]:5678".
func parseAddr(s string) netip.Addr {
	if host, _, err := net.SplitHostPort(s); err == nil {
		s = host
	}
	a, err := netip.ParseAddr(s)
	if err != nil {
		return netip.Addr{}
	}
	return a.Unmap()
}

func isTrusted(a netip.Addr, trusted []netip.Prefix) bool {
	for _, p := range trusted {
		if p.Contains(a) {
			return true
		}
	}
	return false
}
