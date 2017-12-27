package middlewares

import (
	"testing"
	"time"
	"net/http/httptest"
	"net/http"
	"github.com/stretchr/testify/assert"
	. "stash.ges.symantec.com/scm/oasis-connect"
	"bytes"
	"context"
	"sync"
)

func mockRateLimitFunc(max int64, duration time.Duration) RateLimitFunc{
	return func(r *http.Request) []RateLimitQuery {
		return []RateLimitQuery{
			RateLimitQuery{
				"APP_ID.payload" ,
				BodySize(r),
				max,
				duration,
			},
		}
	}
}

func TestHttpRateLimiter_ReturnANewHandler(t *testing.T) {
	handler := NewHttpRateLimiter(mockRateLimitFunc(10, time.Second)).Handler(AlwaysReturn200Handler())
	r, _ := http.NewRequest("GET", "/foo", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	assert.Equal(t, w.Code, http.StatusOK)
}

func TestHttpRateLimiter_RateLimitRequest(t *testing.T) {
	handler := NewHttpRateLimiter(mockRateLimitFunc(10, 100*time.Millisecond)).Handler(AlwaysReturn200Handler())
	w := httptest.NewRecorder()
	r, _ := http.NewRequest("POST", "/foo", bytes.NewBufferString("1234567890"))
	handler.ServeHTTP(w, r)
	assert.Equal(t, w.Code, http.StatusOK, "should not be limitted")

	w = httptest.NewRecorder()
	r, _ = http.NewRequest("POST", "/bar", bytes.NewBufferString("1"))
	handler.ServeHTTP(w, r)
	assert.Equal(t, w.Code, http.StatusTooManyRequests, "should be limitted")
}

func TestHttpRateLimiter_ShouldResetLimitAfterAWhile(t *testing.T) {
	handler := NewHttpRateLimiter(mockRateLimitFunc(1, 100*time.Millisecond)).Handler(AlwaysReturn200Handler())
	w := httptest.NewRecorder()
	r, _ := http.NewRequest("POST", "/foo", bytes.NewBufferString("12"))
	handler.ServeHTTP(w, r)
	assert.Equal(t, w.Code, http.StatusTooManyRequests, "should be limitted")
	time.Sleep(100*time.Millisecond)

	w = httptest.NewRecorder()
	r, _ = http.NewRequest("POST", "/bar", bytes.NewBufferString("1"))
	handler.ServeHTTP(w, r)
	assert.Equal(t, w.Code, http.StatusOK, "should not be limitted")
}

func TestHttpRateLimiter_ShouldNotRateLimitIfDefaultRateIsSet(t *testing.T) {
	handler := NewHttpRateLimiter(mockRateLimitFunc(0, 100*time.Millisecond)).Handler(AlwaysReturn200Handler())
	w := httptest.NewRecorder()
	r, _ := http.NewRequest("POST", "/foo", bytes.NewBufferString("12"))
	handler.ServeHTTP(w, r)
	assert.Equal(t, w.Code, http.StatusOK, "should not be limitted")
}


func TestDefaultHttpRateLimiter_RateLimitOnPayload(t *testing.T) {
	appInfo := AppInfo{
		AppId: "APP_ID",
		MaxBytesPerSecond: 1,
		MaxRequestsPerSecond: 10,
	}
	handler := DefaultHttpRateLimiter().Handler(AlwaysReturn200Handler())
	w := httptest.NewRecorder()
	r, _ := http.NewRequest("POST", "/foo", bytes.NewBufferString("1"))
	r = r.WithContext(context.WithValue(r.Context(), "AppInfo", appInfo))

	handler.ServeHTTP(w, r)
	assert.Equal(t, w.Code, http.StatusOK, "should not be limitted")

	w = httptest.NewRecorder()
	r, _ = http.NewRequest("POST", "/foo", bytes.NewBufferString("1"))
	r = r.WithContext(context.WithValue(r.Context(), "AppInfo", appInfo))

	handler.ServeHTTP(w, r)
	assert.Equal(t, w.Code, http.StatusTooManyRequests, "should be limitted")
}

func TestDefaultHttpRateLimiter_RateLimitOnNumberOfRequests(t *testing.T) {
	appInfo := AppInfo{
		AppId: "APP_ID",
		MaxBytesPerSecond: 0,
		MaxRequestsPerSecond: 5,
	}
	handler := DefaultHttpRateLimiter().Handler(AlwaysReturn200Handler())

	var i int
	for i = 0; i<5; i++ {
		w := httptest.NewRecorder()
		r, _ := http.NewRequest("POST", "/foo", bytes.NewBufferString("1"))
		r = r.WithContext(context.WithValue(r.Context(), "AppInfo", appInfo))
		handler.ServeHTTP(w, r)
		assert.Equal(t, w.Code, http.StatusOK, "should not be limitted")
	}

	w := httptest.NewRecorder()
	r, _ := http.NewRequest("POST", "/foo", bytes.NewBufferString("1"))
	r = r.WithContext(context.WithValue(r.Context(), "AppInfo", appInfo))

	handler.ServeHTTP(w, r)
	assert.Equal(t, w.Code, http.StatusTooManyRequests, "should be limitted")
}


func TestRaceHttpRateLimiter(t *testing.T) {
	handler := NewHttpRateLimiter(mockRateLimitFunc(10, time.Minute)).Handler(AlwaysReturn200Handler())
	N := 4000
	wg := sync.WaitGroup{}
	wg.Add(N)
	for i :=0;i<N;i++ {
		go func() {
			w := httptest.NewRecorder()
			r, _ := http.NewRequest("POST", "/foo", bytes.NewBufferString("1"))
			handler.ServeHTTP(w, r)
			wg.Done()
		}()
	}
	wg.Wait()
}
