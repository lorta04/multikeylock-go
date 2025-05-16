package multikeylock

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestTryLock_Success(t *testing.T) {
	m := New()
	lock, ok := m.TryLock("testkey")
	if !ok {
		t.Fatal("expected to acquire lock")
	}
	defer lock.Unlock()
}

func TestTryLock_Timeout(t *testing.T) {
	m := New(Config{Timeout: 100 * time.Millisecond, Retry: 10 * time.Millisecond})

	lock1, ok := m.TryLock("samekey")
	if !ok {
		t.Fatal("expected to acquire first lock")
	}
	defer lock1.Unlock()

	start := time.Now()
	_, ok = m.TryLock("samekey")
	if ok {
		t.Fatal("should not acquire lock while already held")
	}
	if time.Since(start) < 100*time.Millisecond {
		t.Fatal("timeout was too short")
	}
}

func TestLockCtx_Cancel(t *testing.T) {
	m := New()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	m.TryLock("ctxkey") // Lock once, don't unlock

	_, err := m.LockCtx(ctx, "ctxkey", 5*time.Millisecond)
	if err == nil {
		t.Fatal("expected context timeout")
	}
}

func TestConcurrency_SingleKey(t *testing.T) {
	m := New()

	const goroutines = 10
	key := "concurrent"

	var wg sync.WaitGroup
	wg.Add(goroutines)

	successCount := 0

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()

			lock, ok := m.TryLock(key)
			if !ok {
				t.Errorf("goroutine %d failed to acquire the lock", id)
				return
			}
			defer lock.Unlock()

			time.Sleep(20 * time.Millisecond)

			successCount++
		}(i)
	}

	wg.Wait()

	if successCount != goroutines {
		t.Errorf("expected %d successful locks, got %d", goroutines, successCount)
	} else {
		t.Logf("all %d goroutines acquired and released the lock sequentially", goroutines)
	}
}

func TestConcurrency_MultipleKeys(t *testing.T) {
	m := New(Config{Timeout: 10 * time.Millisecond})

	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)

	var successCount atomic.Int32

	start := time.Now()

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()

			key := fmt.Sprintf("key-%d", id)
			lock, ok := m.TryLock(key)
			if !ok {
				t.Errorf("goroutine %d failed to acquire lock for key %s", id, key)
				return
			}
			defer lock.Unlock()

			time.Sleep(200 * time.Millisecond)

			successCount.Add(1)
		}(i)
	}

	wg.Wait()

	elapsed := time.Since(start)

	if count := successCount.Load(); count != goroutines {
		t.Errorf("expected %d successful locks, got %d", goroutines, count)
	}

	if elapsed > 250*time.Millisecond {
		t.Errorf("expected test to complete within 300ms, took %v", elapsed)
	} else {
		t.Logf("completed in %v with all %d locks acquired", elapsed, successCount.Load())
	}
}

func TestUnlock_WithFakeKeyLock_DoesNotDeleteLock(t *testing.T) {
	m := New()

	lock, ok := m.TryLock("lock")
	if !ok {
		t.Fatal("expected to acquire real lock")
	}
	defer lock.Unlock()

	forged := &KeyLock{
		mu:    m,
		key:   "lock",
		token: 42,
	}
	forged.Unlock()

	if _, ok := m.locks.Load("lock"); !ok {
		t.Fatal("lock was improperly deleted by forged Unlock()")
	}
}
