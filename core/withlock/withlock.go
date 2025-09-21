// Package withlock makes it easier to use a mutex correctly.
package withlock

import "sync"

func Do(mu sync.Locker, f func()) {
	mu.Lock()
	defer mu.Unlock()
	f()
}

func DoErr(mu sync.Locker, f func() error) error {
	mu.Lock()
	defer mu.Unlock()
	return f()
}
