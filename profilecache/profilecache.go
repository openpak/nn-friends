// Package profilecache tracks how fresh the Mii and name rows nn-friends keeps
// for a PID are. Those rows (wiiu.mii, wiiu.principal_basic_info, 3ds.user_data
// mii_*) are only a cache of what the account core holds, read through the
// nn-account adapter. A row is trusted for TTL; a profile_updated event from the
// core invalidates it at once so the next read fetches the new profile.
package profilecache

import (
	"sync"
	"time"
)

// TTL is how long a cached profile is trusted without a profile_updated event.
const TTL = 5 * time.Minute

// RetryAfter is the pause before asking again after a failed refresh, so a
// down adapter is not hit on every friend-list read.
const RetryAfter = 30 * time.Second

var (
	mu      sync.Mutex
	expires = map[uint32]time.Time{}
	now     = time.Now
)

// Fresh reports whether pid's cached profile can be served without a refresh.
func Fresh(pid uint32) bool {
	mu.Lock()
	defer mu.Unlock()
	exp, ok := expires[pid]
	return ok && now().Before(exp)
}

// Mark records a successful refresh.
func Mark(pid uint32) { set(pid, TTL) }

// MarkFailed records a failed refresh: the stale row is served for RetryAfter.
func MarkFailed(pid uint32) { set(pid, RetryAfter) }

// Invalidate forgets pid so the next read refreshes it.
func Invalidate(pid uint32) {
	mu.Lock()
	delete(expires, pid)
	mu.Unlock()
}

func set(pid uint32, d time.Duration) {
	mu.Lock()
	expires[pid] = now().Add(d)
	mu.Unlock()
}
