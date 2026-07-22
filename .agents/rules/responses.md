---
trigger: always_on
---

All API responses should be consistent.

Prefer

success
message
data
errors

Return meaningful HTTP status codes.

Never expose internal errors.