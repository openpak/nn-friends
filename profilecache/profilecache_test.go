package profilecache

import (
	"testing"
	"time"
)

func TestFreshness(t *testing.T) {
	t0 := time.Now()
	now = func() time.Time { return t0 }
	defer func() { now = time.Now }()
	if Fresh(7) {
		t.Fatal("unknown pid must not be fresh")
	}
	Mark(7)
	if !Fresh(7) {
		t.Fatal("marked pid must be fresh")
	}
	now = func() time.Time { return t0.Add(TTL + time.Second) }
	if Fresh(7) {
		t.Fatal("expired after TTL")
	}
	Mark(7)
	Invalidate(7)
	if Fresh(7) {
		t.Fatal("invalidated pid must refresh")
	}
	MarkFailed(8)
	now = func() time.Time { return t0.Add(TTL + RetryAfter + time.Second) }
	if Fresh(8) {
		t.Fatal("failed refresh retries after RetryAfter")
	}
}
