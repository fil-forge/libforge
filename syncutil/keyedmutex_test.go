package syncutil

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeyedMutex(t *testing.T) {
	var l KeyedMutex
	var wg sync.WaitGroup
	var mu sync.Mutex
	held := map[string]int{}
	for i := range 100 {
		key := []string{"a", "b"}[i%2]
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer l.Lock(key)()
			mu.Lock()
			held[key]++
			assert.Equal(t, 1, held[key], "one holder per key")
			mu.Unlock()
			mu.Lock()
			held[key]--
			mu.Unlock()
		}()
	}
	wg.Wait()
	require.Empty(t, l.locks, "unused locks are dropped")
}

// TestKeyedMutexIsKeyed pins the per-key contract: while a is held, a second
// a waits, b is free at once, and the waiting a proceeds only once a is
// released.
func TestKeyedMutexIsKeyed(t *testing.T) {
	var l KeyedMutex
	unlockA := l.Lock("a")

	acquiredA := make(chan struct{})
	go func() {
		defer l.Lock("a")()
		close(acquiredA)
	}()
	select {
	case <-acquiredA:
		t.Fatal("a second a was acquired while the first was held")
	case <-time.After(50 * time.Millisecond):
	}

	done := make(chan struct{})
	go func() {
		defer l.Lock("b")()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("b waited on a")
	}

	unlockA()
	select {
	case <-acquiredA:
	case <-time.After(time.Second):
		t.Fatal("the second a never proceeded after the first was released")
	}
}

// TestUnlockIsIdempotent: a second call of an unlock changes nothing, so the
// key stays held by whoever holds it next and the records stay consistent.
func TestUnlockIsIdempotent(t *testing.T) {
	var l KeyedMutex
	unlock := l.Lock("k")
	unlock()
	unlock2 := l.Lock("k")
	unlock() // a stale second call: must not release the new holder

	acquired := make(chan struct{})
	released := make(chan struct{})
	go func() {
		unlock := l.Lock("k")
		close(acquired)
		unlock()
		close(released)
	}()
	select {
	case <-acquired:
		t.Fatal("a stale unlock released the key")
	case <-time.After(50 * time.Millisecond):
	}
	unlock2()
	select {
	case <-released:
	case <-time.After(time.Second):
		t.Fatal("the key was never released")
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	require.Empty(t, l.locks, "unused locks are dropped")
}
