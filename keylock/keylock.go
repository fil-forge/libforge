// Package keylock serializes work on a key within one process.
package keylock

import "sync"

// Locks holds one mutex per key in use. The zero value is ready to use. A
// Locks must not be copied after first use, as a sync.Mutex must not: a
// copy would share the map and the per-key records while guarding them
// with a mutex of its own.
type Locks struct {
	mu    sync.Mutex
	locks map[string]*lock
}

type lock struct {
	sync.Mutex
	// waiters counts the holder and everyone waiting; the lock is dropped
	// from the map when it reaches zero.
	waiters int
}

// Lock locks key and returns the unlock. The unlock releases the key once;
// calling it again does nothing, so a deferred unlock beside an explicit
// one is safe.
func (l *Locks) Lock(key string) func() {
	l.mu.Lock()
	if l.locks == nil {
		l.locks = map[string]*lock{}
	}
	k, ok := l.locks[key]
	if !ok {
		k = &lock{}
		l.locks[key] = k
	}
	k.waiters++
	l.mu.Unlock()

	k.Lock()
	var once sync.Once
	return func() {
		once.Do(func() {
			k.Unlock()
			l.mu.Lock()
			k.waiters--
			if k.waiters == 0 {
				delete(l.locks, key)
			}
			l.mu.Unlock()
		})
	}
}
