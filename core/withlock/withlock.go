package withlock

import "sync"

func Do(mu *sync.Mutex, f func()) {
	mu.Lock()
	defer mu.Unlock()
	f()
}

func DoErr(mu *sync.Mutex, f func() error) error {
	mu.Lock()
	defer mu.Unlock()
	return f()
}
