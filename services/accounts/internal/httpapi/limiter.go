package httpapi

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// maxLimiterKeys caps how many keys the limiter tracks. Without a cap, an attacker who
// rotates keys (for example source addresses or emails) could grow memory without bound.
const maxLimiterKeys = 100_000

// idleLimiterTTL is how long an unused key is kept before it can be evicted.
const idleLimiterTTL = time.Hour

// limiterEntry holds one key's token bucket and the last time it was used.
type limiterEntry struct {
	bucket   *rate.Limiter
	lastSeen time.Time
}

// keyedLimiter applies a token bucket per key, such as a client IP or an email address.
type keyedLimiter struct {
	mu      sync.Mutex
	entries map[string]*limiterEntry
	limit   rate.Limit
	burst   int
	now     func() time.Time
}

func newKeyedLimiter(limit rate.Limit, burst int, now func() time.Time) *keyedLimiter {
	return &keyedLimiter{
		entries: make(map[string]*limiterEntry),
		limit:   limit,
		burst:   burst,
		now:     now,
	}
}

// Allow reports whether a request for key may proceed, and consumes one token if so.
func (l *keyedLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	entry, ok := l.entries[key]
	if !ok {
		if len(l.entries) >= maxLimiterKeys {
			l.evictIdle(now)
		}
		entry = &limiterEntry{bucket: rate.NewLimiter(l.limit, l.burst)}
		l.entries[key] = entry
	}
	entry.lastSeen = now
	return entry.bucket.Allow()
}

// evictIdle removes keys that have not been used for idleLimiterTTL. If the map is still full,
// it is cleared. Clearing gives every client a fresh bucket, which is a coarse but bounded
// fallback that only matters under a key-rotation attack.
func (l *keyedLimiter) evictIdle(now time.Time) {
	for key, entry := range l.entries {
		if now.Sub(entry.lastSeen) > idleLimiterTTL {
			delete(l.entries, key)
		}
	}
	if len(l.entries) >= maxLimiterKeys {
		l.entries = make(map[string]*limiterEntry)
	}
}
