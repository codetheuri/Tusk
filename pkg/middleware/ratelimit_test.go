package middleware

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
	"time"
)

func TestRateLimiter(t *testing.T) {
	dl := newDummyLogger()
	// Allow 2 requests, refill 0.1 tokens/sec
	limiter := NewRateLimiter(2, 0.1, dl)

	handler := limiter.Limit()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req1 := httptest.NewRequest("GET", "/test", nil)
	req1.RemoteAddr = "192.168.1.1:1234"
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Errorf("req 1 expected 200 OK, got %d", rec1.Code)
	}

	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = "192.168.1.1:1234"
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Errorf("req 2 expected 200 OK, got %d", rec2.Code)
	}

	// 3rd request should be blocked (429 Too Many Requests)
	req3 := httptest.NewRequest("GET", "/test", nil)
	req3.RemoteAddr = "192.168.1.1:1234"
	rec3 := httptest.NewRecorder()
	handler.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusTooManyRequests {
		t.Errorf("req 3 expected 429 Too Many Requests, got %d", rec3.Code)
	}
}

// limited serves through a limiter of burst 1 that never refills, so the
// second request from one caller is refused.
func limited(t *testing.T, opts ...RateLimitOption) func(remote string, xff ...string) int {
	t.Helper()
	handler := NewRateLimiter(1, 0, newDummyLogger(), opts...).Limit()(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	return func(remote string, xff ...string) int {
		req := httptest.NewRequest("POST", "/api/v1/auth/login", nil)
		req.RemoteAddr = remote
		for _, h := range xff {
			req.Header.Add("X-Forwarded-For", h)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec.Code
	}
}

func mustProxies(t *testing.T, list string) []netip.Prefix {
	t.Helper()
	p, err := ParseTrustedProxies(list)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// Each new connection has a new port; keying on it gave every connection a
// full bucket.
func TestRateLimiterIgnoresThePort(t *testing.T) {
	do := limited(t)
	if got := do("41.90.1.2:50001"); got != http.StatusOK {
		t.Fatalf("first: %d", got)
	}
	if got := do("41.90.1.2:50002"); got != http.StatusTooManyRequests {
		t.Fatalf("same address, new port: got %d, want 429", got)
	}
}

// Without a trusted proxy, X-Forwarded-For is whatever the client sent.
func TestRateLimiterIgnoresForwardedForFromAnyone(t *testing.T) {
	do := limited(t)
	do("41.90.1.2:50001", "1.1.1.1")
	if got := do("41.90.1.2:50002", "2.2.2.2"); got != http.StatusTooManyRequests {
		t.Fatalf("a made-up X-Forwarded-For bought a new bucket: got %d", got)
	}
}

// Behind the proxy, each client is its own caller, and what a client puts in
// front of the proxy's entry changes nothing.
func TestRateLimiterBehindATrustedProxy(t *testing.T) {
	do := limited(t, WithTrustedProxies(mustProxies(t, "172.16.0.0/12")))
	const proxy = "172.20.0.5:40000"

	if got := do(proxy, "41.90.1.2"); got != http.StatusOK {
		t.Fatalf("client A: %d", got)
	}
	if got := do(proxy, "41.90.9.9"); got != http.StatusOK {
		t.Fatalf("client B is a separate caller: %d", got)
	}
	if got := do(proxy, "6.6.6.6, 41.90.1.2"); got != http.StatusTooManyRequests {
		t.Fatalf("client A with a forged entry in front: got %d, want 429", got)
	}
	// The same, spread over two headers.
	if got := do(proxy, "7.7.7.7", "41.90.1.2"); got != http.StatusTooManyRequests {
		t.Fatalf("client A with a forged header in front: got %d, want 429", got)
	}
}

func TestClientIP(t *testing.T) {
	trusted := mustProxies(t, "10.0.0.0/8, 172.20.0.40")
	cases := []struct {
		name, remote string
		xff          []string
		want         string
	}{
		{"direct", "41.90.1.2:1", nil, "41.90.1.2"},
		{"direct, header ignored", "41.90.1.2:1", []string{"1.1.1.1"}, "41.90.1.2"},
		{"one trusted hop", "172.20.0.40:1", []string{"41.90.1.2"}, "41.90.1.2"},
		{"two trusted hops", "10.0.0.1:1", []string{"9.9.9.9, 41.90.1.2, 172.20.0.40"}, "41.90.1.2"},
		{"trusted, no header", "172.20.0.40:1", nil, "172.20.0.40"},
		{"garbage stops the walk", "172.20.0.40:1", []string{"41.90.1.2, nonsense"}, "172.20.0.40"},
		{"ipv6 with port", "[2001:db8::1]:443", nil, "2001:db8::1"},
		{"ipv4-mapped", "[::ffff:41.90.1.2]:1", nil, "41.90.1.2"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = c.remote
			for _, h := range c.xff {
				req.Header.Add("X-Forwarded-For", h)
			}
			if got := ClientIP(req, trusted).String(); got != c.want {
				t.Fatalf("got %s, want %s", got, c.want)
			}
		})
	}
}

// One device can take any address in its /64.
func TestRateLimiterGroupsIPv6ByItsSlash64(t *testing.T) {
	do := limited(t)
	do("[2001:db8:1:2::1]:1")
	if got := do("[2001:db8:1:2:ffff::9]:1"); got != http.StatusTooManyRequests {
		t.Fatalf("same /64: got %d, want 429", got)
	}
	if got := do("[2001:db8:1:3::1]:1"); got != http.StatusOK {
		t.Fatalf("another /64: got %d", got)
	}
}

// A flood of fresh addresses cannot grow the table without bound, nor buy a
// bucket each once it is full.
func TestRateLimiterCapsWhatItTracks(t *testing.T) {
	rl := NewRateLimiter(1, 0, newDummyLogger(), WithMaxClients(2))
	if !rl.Allow("a") || !rl.Allow("b") {
		t.Fatal("the first two callers")
	}
	if !rl.Allow("c") {
		t.Fatal("the first caller past the cap takes the shared bucket")
	}
	if rl.Allow("d") {
		t.Fatal("the next new caller shares that same, now empty, bucket")
	}
	if n := len(rl.visitors); n > 3 {
		t.Fatalf("tracking %d callers, want at most the cap and the shared bucket", n)
	}
}

// A caller whose bucket has filled up again is the same as a new one, so it
// makes room for new callers before anyone is sent to the shared bucket.
func TestRateLimiterDropsRefilledCallersWhenFull(t *testing.T) {
	rl := NewRateLimiter(1, 1000, newDummyLogger(), WithMaxClients(1))
	rl.Allow("a")
	time.Sleep(5 * time.Millisecond) // a's bucket refills
	rl.Allow("b")
	if _, ok := rl.visitors[overflowKey]; ok {
		t.Fatal("b was sent to the shared bucket although a had refilled")
	}
}

func TestParseTrustedProxies(t *testing.T) {
	got, err := ParseTrustedProxies(" 172.16.0.0/12, 127.0.0.1 ,,::1")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"172.16.0.0/12", "127.0.0.1/32", "::1/128"}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for i := range want {
		if got[i].String() != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
	if _, err := ParseTrustedProxies("172.16.0.0/40"); err == nil {
		t.Fatal("a bad range was accepted")
	}
	if _, err := ParseTrustedProxies("proxy.local"); err == nil {
		t.Fatal("a name was accepted")
	}
}
