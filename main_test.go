package main

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// The limiter has to hold the number of concurrent handlers at the limit, and
// let every request through eventually rather than refusing the overflow.
func TestLimitConcurrency(t *testing.T) {
	const (
		limit    = 2
		requests = 10
	)

	var (
		inFlight int32
		peak     int32
		release  = make(chan struct{})
	)

	handler := limitConcurrency(limit)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		now := atomic.AddInt32(&inFlight, 1)
		for {
			seen := atomic.LoadInt32(&peak)
			if now <= seen || atomic.CompareAndSwapInt32(&peak, seen, now) {
				break
			}
		}

		<-release
		atomic.AddInt32(&inFlight, -1)
		w.WriteHeader(http.StatusOK)
	}))

	var wg sync.WaitGroup

	codes := make([]int, requests)

	for i := range codes {
		wg.Add(1)

		// `go.mod` still asks for 1.21 semantics, where the loop variable is
		// shared across iterations.
		go func(i int) {
			defer wg.Done()

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
			codes[i] = rec.Code
		}(i)
	}

	// Let the first batch pile up against the limit before draining.
	assert.Eventually(t, func() bool {
		return atomic.LoadInt32(&inFlight) == limit
	}, time.Second, time.Millisecond, "the limit should be reached")

	close(release)
	wg.Wait()

	assert.Equal(t, int32(limit), atomic.LoadInt32(&peak),
		"no more than the limit should run at once")

	for i, code := range codes {
		assert.Equal(t, http.StatusOK, code, "request %d should have been served", i)
	}
}

func TestLimitConcurrencyDisabled(t *testing.T) {
	handler := limitConcurrency(0)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))

	assert.Equal(t, http.StatusOK, rec.Code)
}
