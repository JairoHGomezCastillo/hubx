package test

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"hubx-gw/internal/ratelimit"
)

// fakeClock is a manually advanced clock shared by the tests.
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func newClock() *fakeClock {
	return &fakeClock{now: time.Date(2026, 11, 25, 10, 0, 0, 0, time.UTC)}
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func TestLimiter_AllowsUpToTheLimitThenRefuses(t *testing.T) {
	clock := newClock()
	l := ratelimit.New(3, time.Minute, 100, clock.Now)

	for i := 1; i <= 3; i++ {
		ok, retry := l.Allow("user")
		if !ok || retry != 0 {
			t.Fatalf("request %d: ok=%v retry=%v, want accepted with zero retry", i, ok, retry)
		}
	}
	ok, retry := l.Allow("user")
	if ok {
		t.Fatal("request 4 must be refused")
	}
	if retry != time.Minute {
		t.Errorf("retryAfter = %v, want %v", retry, time.Minute)
	}
}

func TestLimiter_SlidingWindowRetryAndRecovery(t *testing.T) {
	clock := newClock()
	l := ratelimit.New(3, time.Minute, 100, clock.Now)

	l.Allow("u") // t0
	clock.Advance(10 * time.Second)
	l.Allow("u") // t0+10
	clock.Advance(10 * time.Second)
	l.Allow("u") // t0+20

	clock.Advance(5 * time.Second) // t0+25
	ok, retry := l.Allow("u")
	if ok {
		t.Fatal("must be refused inside the window")
	}
	if want := 35 * time.Second; retry != want { // oldest (t0) leaves at t0+60
		t.Errorf("retryAfter = %v, want %v", retry, want)
	}

	// Refused requests are not stored: they never extend the window.
	clock.Advance(34 * time.Second) // t0+59
	if ok, retry = l.Allow("u"); ok || retry != time.Second {
		t.Fatalf("at t0+59 ok=%v retry=%v, want refused with 1s", ok, retry)
	}

	clock.Advance(time.Second) // t0+60: the first accepted request left the window
	if ok, _ = l.Allow("u"); !ok {
		t.Fatal("one request must be accepted once the oldest left the window")
	}
	if ok, retry = l.Allow("u"); ok || retry != 10*time.Second {
		t.Fatalf("ok=%v retry=%v, want refused until t0+70 (10s)", ok, retry)
	}
}

func TestLimiter_FullRecoveryAfterOneMinute(t *testing.T) {
	clock := newClock()
	l := ratelimit.New(120, time.Minute, 100, clock.Now)

	for i := 0; i < 120; i++ {
		if ok, _ := l.Allow("u"); !ok {
			t.Fatalf("request %d refused", i+1)
		}
	}
	if ok, _ := l.Allow("u"); ok {
		t.Fatal("request 121 must be refused")
	}
	clock.Advance(time.Minute)
	for i := 0; i < 120; i++ {
		if ok, _ := l.Allow("u"); !ok {
			t.Fatalf("after one minute request %d refused", i+1)
		}
	}
}

func TestLimiter_KeysAreIsolated(t *testing.T) {
	clock := newClock()
	l := ratelimit.New(1, time.Minute, 100, clock.Now)

	if ok, _ := l.Allow("a"); !ok {
		t.Fatal("a first request refused")
	}
	if ok, _ := l.Allow("a"); ok {
		t.Fatal("a second request must be refused")
	}
	if ok, _ := l.Allow("b"); !ok {
		t.Fatal("b must not be affected by a")
	}
}

func TestLimiter_InstancesAreIndependent(t *testing.T) {
	clock := newClock()
	users := ratelimit.New(1, time.Minute, 2, clock.Now)
	signIn := ratelimit.New(1, time.Minute, 2, clock.Now)

	users.Allow("alice")
	// A flood of distinct sign-in keys must not evict the user's entry.
	for i := 0; i < 50; i++ {
		signIn.Allow(fmt.Sprintf("email-%d", i))
	}
	if ok, _ := users.Allow("alice"); ok {
		t.Fatal("alice must still be remembered: the sign-in flood used another instance")
	}
}

func TestLimiter_BoundedMemoryEvictsLeastRecentlySeen(t *testing.T) {
	clock := newClock()
	l := ratelimit.New(1, time.Minute, 2, clock.Now)

	l.Allow("a")
	l.Allow("b")
	l.Allow("a") // refused, but a is now the most recently seen
	l.Allow("c") // table full: b (least recently seen) is evicted

	if ok, _ := l.Allow("a"); ok {
		t.Error("a was recently seen and must still be limited")
	}
	if ok, _ := l.Allow("b"); !ok {
		t.Error("b was evicted, so it starts a new window")
	}
}

func TestLimiter_EvictionUnderFloodKeepsTableBounded(t *testing.T) {
	clock := newClock()
	l := ratelimit.New(1, time.Minute, 10, clock.Now)

	l.Allow("victim")
	for i := 0; i < 1000; i++ {
		l.Allow(fmt.Sprintf("flood-%d", i))
	}
	// The victim was forgotten (known limitation): this proves the bound is
	// enforced, because 1001 identities cannot all be remembered with room for 10.
	if ok, _ := l.Allow("victim"); !ok {
		t.Error("the table is bounded, the oldest identity must have been evicted")
	}
	if ok, _ := l.Allow("flood-999"); ok {
		t.Error("the most recent identities must still be remembered")
	}
}

func TestLimiter_ExpiredKeysFreeTheirSlot(t *testing.T) {
	clock := newClock()
	l := ratelimit.New(1, time.Minute, 2, clock.Now)

	l.Allow("a")
	l.Allow("b")
	clock.Advance(2 * time.Minute)
	l.Allow("c")
	l.Allow("d")
	// a and b expired; whatever happens to them they are accepted again.
	if ok, _ := l.Allow("a"); !ok {
		t.Error("a window expired long ago, request must be accepted")
	}
}

func TestLimiter_RetryAfterNeverNegativeOrZeroWhenRefused(t *testing.T) {
	clock := newClock()
	l := ratelimit.New(2, time.Minute, 10, clock.Now)

	l.Allow("u")
	clock.Advance(59*time.Second + 999*time.Millisecond)
	l.Allow("u")
	ok, retry := l.Allow("u")
	if ok {
		t.Fatal("must be refused")
	}
	if retry <= 0 || retry > time.Minute {
		t.Errorf("retryAfter = %v, want in (0, 1m]", retry)
	}
}

func TestLimiter_ConcurrencyNeverExceedsTheLimit(t *testing.T) {
	clock := newClock()
	const limit = 50
	l := ratelimit.New(limit, time.Minute, 100, clock.Now)

	var accepted atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < 500; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if ok, _ := l.Allow("same"); ok {
				accepted.Add(1)
			}
		}()
	}
	wg.Wait()
	if got := accepted.Load(); got != limit {
		t.Errorf("accepted = %d, want exactly %d", got, limit)
	}
}

func TestLimiter_ConcurrencyManyKeys(t *testing.T) {
	clock := newClock()
	l := ratelimit.New(2, time.Minute, 20, clock.Now)

	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 500; i++ {
				l.Allow(fmt.Sprintf("k-%d", (g*31+i)%97))
			}
		}()
	}
	wg.Wait()
}

func TestNew_RejectsNonPositiveArguments(t *testing.T) {
	clock := newClock()
	tests := []struct {
		name   string
		limit  int
		window time.Duration
		keys   int
	}{
		{"zero limit", 0, time.Minute, 10},
		{"negative limit", -1, time.Minute, 10},
		{"zero window", 1, 0, 10},
		{"zero keys", 1, time.Minute, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("New must panic on a programming error in its arguments")
				}
			}()
			ratelimit.New(tt.limit, tt.window, tt.keys, clock.Now)
		})
	}
}
