// Package ratelimit implements the in-memory sliding-window limiter used by the
// gateway. State is per process: it resets on restart and every gateway
// instance counts separately (documented limitation of the story).
package ratelimit

import (
	"container/list"
	"sync"
	"time"
)

//go:generate go tool mockgen -source=limiter.go -destination=mocks/limiter_mock.go -package=mocks

// Limiter decides whether a request identified by key may proceed.
type Limiter interface {
	// Allow records the request when it is accepted. When it is refused it
	// returns how long until the oldest accepted request leaves the window;
	// refused requests are never stored, so they do not extend the window.
	Allow(key string) (ok bool, retryAfter time.Duration)
}

type entry struct {
	key        string
	timestamps []time.Time // accepted requests only, oldest first, at most limit
}

// SlidingWindow is a Limiter with a sliding window and a bounded number of
// remembered keys (least recently seen evicted first). It starts no goroutine.
type SlidingWindow struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	maxKeys int
	now     func() time.Time
	order   *list.List // front = most recently seen
	items   map[string]*list.Element
}

// New returns a Limiter that accepts at most limit requests per key within
// window and remembers at most maxKeys keys. now is the injected clock.
// It panics if limit, window or maxKeys is not positive or now is nil: they are
// programming errors, configuration is validated before reaching here.
func New(limit int, window time.Duration, maxKeys int, now func() time.Time) Limiter {
	if limit <= 0 || window <= 0 || maxKeys <= 0 || now == nil {
		panic("ratelimit: limit, window, maxKeys and clock must be positive and set")
	}
	return &SlidingWindow{
		limit:   limit,
		window:  window,
		maxKeys: maxKeys,
		now:     now,
		order:   list.New(),
		items:   make(map[string]*list.Element),
	}
}

// Allow implements Limiter.
func (s *SlidingWindow) Allow(key string) (bool, time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	e := s.touch(key)

	// Drop accepted requests that already left the window.
	cutoff := now.Add(-s.window)
	drop := 0
	for drop < len(e.timestamps) && !e.timestamps[drop].After(cutoff) {
		drop++
	}
	if drop > 0 {
		e.timestamps = append(e.timestamps[:0], e.timestamps[drop:]...)
	}

	if len(e.timestamps) >= s.limit {
		return false, e.timestamps[0].Add(s.window).Sub(now)
	}
	e.timestamps = append(e.timestamps, now)
	return true, 0
}

// touch returns the entry of key marked as most recently seen, creating it and
// evicting the least recently seen one when the table is full. Caller holds mu.
func (s *SlidingWindow) touch(key string) *entry {
	if el, ok := s.items[key]; ok {
		s.order.MoveToFront(el)
		return el.Value.(*entry)
	}
	for s.order.Len() >= s.maxKeys {
		oldest := s.order.Back()
		s.order.Remove(oldest)
		delete(s.items, oldest.Value.(*entry).key)
	}
	e := &entry{key: key}
	s.items[key] = s.order.PushFront(e)
	return e
}
