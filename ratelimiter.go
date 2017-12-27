package middlewares

import (
	"time"
	"sync"
	"golang.org/x/time/rate"
	"strconv"
	"strings"
)

// NewLimiter is a constructor for Limiter.
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		tokenBuckets: make(map[string]*rate.Limiter),
	}
}

// Limiter is a config struct to limit a particular request handler.
type RateLimiter struct {
	tokenBuckets map[string]*rate.Limiter
	sync.RWMutex
}

// LimitReached returns a bool indicating if the Bucket identified by key ran out of tokens.
func (l *RateLimiter) LimitReached(q RateLimitQuery) bool {
	key := q.Key + ":" + q.Duration.String() + ":" + strconv.FormatInt(q.Max, 10)
	l.RLock()
	if _, found := l.tokenBuckets[key]; !found {
		l.RUnlock()
		l.initNewLimiter(key, q)
	} else {
		l.RUnlock()
	}
	l.RLock()
	defer l.RUnlock()
	//rate.Limiter has its own locking mechanism so it's thread safe
	// but we still have to RLock the hashmap before reading
	// otherwise Go will throw fatal when it detects concurrent read& write
	// https://jira.ges.symantec.com/browse/HSELF-1282
	return !l.tokenBuckets[key].AllowN(time.Now(), q.Val)
}

func (l* RateLimiter) initNewLimiter(newKey string, q RateLimitQuery) {
	l.Lock()
	defer l.Unlock()
	// in case we update the rate, need to look for the old key and remove it
	for oldKey, _ := range(l.tokenBuckets) {
		if strings.HasPrefix(oldKey, q.Key + ":") {
			delete(l.tokenBuckets, oldKey)
			break
		}
	}
	l.tokenBuckets[newKey] = rate.NewLimiter(rate.Limit(float64(q.Max) * time.Second.Seconds() / q.Duration.Seconds()), int(q.Max))
}
