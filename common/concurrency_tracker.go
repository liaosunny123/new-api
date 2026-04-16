package common

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

const concurrencyKeyTTL = 5 * time.Minute

type ConcurrencyTracker struct {
	memStore sync.Map // key -> *int64
}

var GlobalConcurrencyTracker = &ConcurrencyTracker{}

// Acquire tries to acquire a concurrency slot.
// Returns (allowed, release). Call release() when the request finishes.
// If limit <= 0, always returns (true, noop).
func (ct *ConcurrencyTracker) Acquire(key string, limit int) (bool, func()) {
	if limit <= 0 {
		return true, func() {}
	}

	if RedisEnabled {
		return ct.acquireRedis(key, limit)
	}
	return ct.acquireMemory(key, limit)
}

// GetCurrentCount returns the current concurrency count for the given key.
func (ct *ConcurrencyTracker) GetCurrentCount(key string) int64 {
	if RedisEnabled {
		ctx := context.Background()
		val, err := RDB.Get(ctx, key).Int64()
		if err != nil {
			return 0
		}
		return val
	}

	if ptr, ok := ct.memStore.Load(key); ok {
		return atomic.LoadInt64(ptr.(*int64))
	}
	return 0
}

func (ct *ConcurrencyTracker) acquireRedis(key string, limit int) (bool, func()) {
	ctx := context.Background()

	// INCR atomically and check
	val, err := RDB.Incr(ctx, key).Result()
	if err != nil {
		// On Redis error, allow the request (fail-open)
		SysLog("concurrency tracker Redis INCR error: " + err.Error())
		return true, func() {}
	}

	// Set TTL as safety net (only on first increment or refresh)
	RDB.Expire(ctx, key, concurrencyKeyTTL)

	if val > int64(limit) {
		// Over limit - decrement back
		RDB.Decr(ctx, key)
		return false, func() {}
	}

	var once sync.Once
	release := func() {
		once.Do(func() {
			RDB.Decr(ctx, key)
		})
	}
	return true, release
}

func (ct *ConcurrencyTracker) acquireMemory(key string, limit int) (bool, func()) {
	var zero int64
	actual, _ := ct.memStore.LoadOrStore(key, &zero)
	ptr := actual.(*int64)

	newVal := atomic.AddInt64(ptr, 1)
	if newVal > int64(limit) {
		atomic.AddInt64(ptr, -1)
		return false, func() {}
	}

	var once sync.Once
	release := func() {
		once.Do(func() {
			atomic.AddInt64(ptr, -1)
		})
	}
	return true, release
}
