package server

import (
	"errors"
	"sync"
)

var ErrNoBackends = errors.New("no backends available")

type BackendPool struct {
	mu       sync.Mutex
	backends []string
	next     int
}

func (p *BackendPool) Next() (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.backends) == 0 {
		return "", ErrNoBackends
	}

	backend := p.backends[p.next]
	p.next = (p.next + 1) % len(p.backends)

	return backend, nil
}

func NewBackendPool(backends []string) *BackendPool {
	copied := append([]string(nil), backends...)

	return &BackendPool{
		backends: copied,
	}
}
