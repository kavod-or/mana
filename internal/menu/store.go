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
	mu              sync.RWMutex
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
	if time.Now().Before(store.nextReloadAfter) {
		config := store.current
		store.mu.Unlock()
		return config, nil
	}
	store.nextReloadAfter = time.Now().Add(store.reloadInterval)
	previous := store.current
	store.mu.Unlock()

	config, err := store.loader()
	if err != nil {
		return previous, err
	}

	store.mu.Lock()
	store.current = config
	store.mu.Unlock()
	return config, nil
}
