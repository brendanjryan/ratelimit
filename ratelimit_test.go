package ratelimit

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/andres-erbsen/clock"
	"github.com/stretchr/testify/assert"
)

// a tiny wrapper around int to make it atomic
type aInt32 struct {
	v int32
}

func (a *aInt32) Add(n int32) int32 {
	return atomic.AddInt32(&a.v, n)
}

func (a *aInt32) Inc() int32 {
	return a.Add(1)
}

func (a *aInt32) Val() int32 {
	return atomic.LoadInt32(&a.v)
}

func TestRateLimiter(t *testing.T) {
	delta := 10.0
	rate := 100
	wg := sync.WaitGroup{}
	wg.Add(1)
	defer wg.Wait()

	c := clock.NewMock()
	rl := New(rate).WithClock(c)

	count := aInt32{}

	done := make(chan struct{})
	defer close(done)

	// spwan jobs
	for i := 0; i < 10; i++ {
		go work(rl, &count, done)
	}

	c.AfterFunc(1*time.Second, func() {
		assert.InDelta(t, rate, count.Val(), delta)
	})

	c.AfterFunc(2*time.Second, func() {
		assert.InDelta(t, 2*rate, count.Val(), delta)
	})

	c.AfterFunc(3*time.Second, func() {
		assert.InDelta(t, 3*rate, count.Val(), delta)
		wg.Done()
	})

	c.Add(5 * time.Second)
	c.Add(10 * time.Second)
}

// a test job that continuously runs until the done channel is closed.
func work(rl Limiter, count *aInt32, done <-chan struct{}) {

	for {
		rl.Take()
		count.Inc()

		select {
		case <-done:
			return
		default:
		}
	}
}
