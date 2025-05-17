package multikeylock

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

var globalTokenId atomic.Int64

type Config struct {
	Timeout time.Duration
	Retry   time.Duration
}

type MultiKeyLock struct {
	locks   sync.Map // map[string]tokenId{} — presence means lock is held
	timeout time.Duration
	retry   time.Duration
}

const (
	defaultTimeout = 5 * time.Second
	defaultRetry   = 10 * time.Millisecond
)

func New(cfg ...Config) *MultiKeyLock {
	c := Config{Timeout: defaultTimeout, Retry: defaultRetry}
	if len(cfg) > 0 {
		if cfg[0].Timeout != 0 {
			c.Timeout = cfg[0].Timeout
		}
		if cfg[0].Retry != 0 {
			c.Retry = cfg[0].Retry
		}
	}
	return &MultiKeyLock{timeout: c.Timeout, retry: c.Retry}
}

func (m *MultiKeyLock) TryLock(key string) (*KeyLock, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()
	return m.lockWithContext(ctx, key, m.retry)
}

func (m *MultiKeyLock) LockCtx(ctx context.Context, key string, retry time.Duration) (*KeyLock, error) {
	kl, ok := m.lockWithContext(ctx, key, retry)
	if !ok {
		return nil, ctx.Err()
	}
	return kl, nil
}

func (m *MultiKeyLock) lockWithContext(ctx context.Context, key string, retry time.Duration) (*KeyLock, bool) {
	ticker := time.NewTicker(retry)
	defer ticker.Stop()

	tokenId := globalTokenId.Add(1)

	for {
		if _, loaded := m.locks.LoadOrStore(key, tokenId); !loaded {
			return &KeyLock{mu: m, key: key, token: tokenId}, true
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
	mu    *MultiKeyLock
	key   string
	token int64
}

func (kl *KeyLock) Unlock() {
	if kl == nil || kl.mu == nil {
		return
	}

	if cur, ok := kl.mu.locks.Load(kl.key); ok {
		if id, ok := cur.(int64); ok && id == kl.token {
			kl.mu.locks.Delete(kl.key)
		}
	}

	kl.mu = nil
}
