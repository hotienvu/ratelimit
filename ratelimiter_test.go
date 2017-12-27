package middlewares

import (
	"testing"
	"time"
	"github.com/stretchr/testify/assert"
	"sync/atomic"
	"sync"
	"math/rand"
	"strconv"
)

func TestRateLimiter_LimitOnKey(t *testing.T) {
	rl := NewRateLimiter()
	assert.False(t, rl.LimitReached(RateLimitQuery{"app", 1, 1, time.Second }))
	assert.True(t, rl.LimitReached(RateLimitQuery{"app", 1, 1, time.Second }))
}

func TestRateLimiter_LimitOnTimeWindow(t *testing.T) {
	rl := NewRateLimiter()
	assert.False(t, rl.LimitReached(RateLimitQuery{"app", 1, 1, 100*time.Millisecond }))
	assert.True(t, rl.LimitReached(RateLimitQuery{"app", 1, 1, 100*time.Millisecond }))
	time.Sleep(100*time.Millisecond)
	assert.False(t, rl.LimitReached(RateLimitQuery{"app", 1, 1, 100*time.Millisecond }))
}

func TestRateLimiter_ResetWhenApplyNewRate(t *testing.T) {
	rl := NewRateLimiter()
	for i:=0;i<10;i++ {
		assert.False(t, rl.LimitReached(RateLimitQuery{"app", 1, 10, time.Second }))
	}
	assert.True(t, rl.LimitReached(RateLimitQuery{"app", 1, 10, time.Second }))
	_, found := rl.tokenBuckets["app:1s:10"]
	assert.True(t, found, "key should be set")

	assert.False(t, rl.LimitReached(RateLimitQuery{"app", 10, 20, time.Second }))
	_, found = rl.tokenBuckets["app:1s:10"]
	assert.False(t, found, "old key should be removed")
	assert.True(t, rl.LimitReached(RateLimitQuery{"app", 11, 20, time.Second }))

	_, found = rl.tokenBuckets["app:1s:20"]
	assert.True(t, found, "new key should be set")
}

func BenchmarkRateLimiter(b *testing.B) {
	rl := NewRateLimiter()
	var success int64 = 0
	wg := new(sync.WaitGroup)
	var max int64 = 100
	wg.Add(b.N)
	for n := 0; n < b.N; n++ {
		go func() {
			if !rl.LimitReached(RateLimitQuery{ strconv.Itoa(rand.Intn(1000)), 1, max, time.Minute }) {
				atomic.AddInt64(&success, 1)
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func TestRaceRateLimiter(t *testing.T) {
	rl := NewRateLimiter()
	var success int64 = 0
	wg := new(sync.WaitGroup)
	N := 1000
	var max int64 = 100
	wg.Add(N)
	for i :=0;i<N;i++ {
		go func() {
			if !rl.LimitReached(RateLimitQuery{"app", 1, max, time.Minute }) {
				atomic.AddInt64(&success, 1)
			}
			wg.Done()
		}()
	}
	wg.Wait()
}