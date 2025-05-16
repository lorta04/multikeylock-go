package multikeylock

import (
	"context"
	"sync"
	"time"
)

type Config struct {
	Timeout time.Duration
	Retry   time.Duration
}

type MultiKeyMutex struct {
	locks   sync.Map // map[string]struct{} — presence means lock is held
	timeout time.Duration
	retry   time.Duration
}

const (
	defaultTimeout = 5 * time.Second
	defaultRetry   = 10 * time.Millisecond
)

var token = struct{}{}

func New(cfg ...Config) *MultiKeyMutex {
	c := Config{Timeout: defaultTimeout, Retry: defaultRetry}
	if len(cfg) > 0 {
		if cfg[0].Timeout != 0 {
			c.Timeout = cfg[0].Timeout
		}
		if cfg[0].Retry != 0 {
			c.Retry = cfg[0].Retry
		}
	}
	return &MultiKeyMutex{timeout: c.Timeout, retry: c.Retry}
}

func (m *MultiKeyMutex) TryLock(key string) (*KeyLock, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()
	return m.lockWithContext(ctx, key, m.retry)
}

func (m *MultiKeyMutex) LockCtx(ctx context.Context, key string, retry time.Duration) (*KeyLock, error) {
	kl, ok := m.lockWithContext(ctx, key, retry)
	if !ok {
		return nil, ctx.Err()
	}
	return kl, nil
}

func (m *MultiKeyMutex) lockWithContext(ctx context.Context, key string, retry time.Duration) (*KeyLock, bool) {
	ticker := time.NewTicker(retry)
	defer ticker.Stop()

	for {
		if _, loaded := m.locks.LoadOrStore(key, token); !loaded {
			return &KeyLock{mu: m, key: key, locked: true}, true
		}

		select {
		case <-ctx.Done():
			return nil, false
		case <-ticker.C:
			// retry
		}
	}
}

type KeyLock struct {
	mu     *MultiKeyMutex
	key    string
	locked bool
}

func (kl *KeyLock) Unlock() {
	if kl == nil || !kl.locked {
		return
	}
	kl.mu.locks.Delete(kl.key)
	kl.locked = false
}
