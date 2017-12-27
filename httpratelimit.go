package middlewares

import (
	"net/http"
	"stash.ges.symantec.com/scm/oasis-connect"
	"time"
	"sync"
)

type RateLimitQuery struct {
	Key string
	Val int
	Max int64
	Duration time.Duration
}

type RateLimitFunc func(*http.Request) []RateLimitQuery

type HttpRateLimiter struct {
	limiter *RateLimiter
	monitFunc RateLimitFunc
	sync.RWMutex
}

func NewHttpRateLimiter(monitFunc RateLimitFunc) *HttpRateLimiter{
	return &HttpRateLimiter{
		limiter: NewRateLimiter(),
		monitFunc: monitFunc,
	}
}

func (rl *HttpRateLimiter) Handler(h http.Handler) http.Handler  {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if rl.limitReached(r) {
			oasis_connect.ReturnErrorResponse(w, http.StatusTooManyRequests, oasis_connect.ResponseMessageTooManyRequests)
		} else {
			h.ServeHTTP(w, r)
		}
	})
}

func (rl *HttpRateLimiter) limitReached(r *http.Request) bool {
	for _, m := range(rl.monitFunc(r)) {
		if m.Max != 0 && rl.limiter.LimitReached(m){
			return true
		}
	}
	return false
}


func DefaultHttpRateLimiter() *HttpRateLimiter {
	rateLimitFunc := func (r *http.Request) []RateLimitQuery {
		appInfo := oasis_connect.AppInfoFromRequest(r)
		return []RateLimitQuery {
			RateLimitQuery{
				appInfo.AppId + ".payload" ,
				oasis_connect.BodySize(r),
				appInfo.MaxBytesPerSecond,
				time.Second,
			},
			RateLimitQuery{
				appInfo.AppId + ".requests" ,
				1,
				appInfo.MaxRequestsPerSecond,
				time.Second,
			},
		}
	}
	return NewHttpRateLimiter(rateLimitFunc)
}

