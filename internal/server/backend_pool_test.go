package server

import (
	"errors"
	"testing"
)

func TestBackendPoolNextUsesRoundRobin(t *testing.T) {

	pool := NewBackendPool([]string{
		"backend-a",
		"backend-b",
		"backend-c",
	})

	expected := []string{
		"backend-a",
		"backend-b",
		"backend-c",
		"backend-a",
		"backend-b",
	}

	for i, want := range expected {
		got, err := pool.Next()
		if err != nil {
			t.Fatalf("Next() call %d: %v", i, err)
		}

		if got != want {
			t.Errorf("Next() call %d = %q, want %q", i, got, want)
		}
	}
}

func TestBackendPoolNextRejectsEmptyPool(t *testing.T) {
	var pool BackendPool

	got, err := pool.Next()

	if !errors.Is(err, ErrNoBackends) {
		t.Fatalf("Next() error = %v, want %v", err, ErrNoBackends)
	}

	if got != "" {
		t.Errorf("Next() = %q, want empty string", got)
	}
}
