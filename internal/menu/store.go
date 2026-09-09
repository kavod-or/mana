package menu

import (
	"sync"
	"time"
)

type Loader func() (Config, error)

// Store reloads the configured menu while retaining the last valid version.
// Reload attempts are rate-limited so a busy conference does not parse the
// same file hundreds of times per second.
type Store struct {
	mu              sync.Mutex
	reloading       bool
	loader          Loader
	current         Config
	reloadInterval  time.Duration
	nextReloadAfter time.Time
}

func NewStore(loader Loader) (*Store, error) {
	return newStore(loader, 500*time.Millisecond)
}

func newStore(loader Loader, reloadInterval time.Duration) (*Store, error) {
	config, err := loader()
	if err != nil {
		return nil, err
	}
	return &Store{
		loader:          loader,
		current:         config,
		reloadInterval:  reloadInterval,
		nextReloadAfter: time.Now().Add(reloadInterval),
	}, nil
}

func (store *Store) Current() (Config, error) {
	store.mu.Lock()
	if store.reloading || time.Now().Before(store.nextReloadAfter) {
		config := store.current
		store.mu.Unlock()
		return config, nil
	}
	store.reloading = true
	store.mu.Unlock()

	config, err := store.loader()
	store.mu.Lock()
	defer store.mu.Unlock()
	store.reloading = false
	store.nextReloadAfter = time.Now().Add(store.reloadInterval)
	if err == nil {
		store.current = config
	}
	return store.current, err
}
