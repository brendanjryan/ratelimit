package ratelimit

import (
	"sync"
	"time"

	"github.com/andres-erbsen/clock"
)

type Limiter interface {
	// will block
	Take() time.Time
}

type Clock interface {
	Now() time.Time
	Sleep(time.Duration)
}

type limiter struct {
	sync.Mutex
	// the time of the last reqeust that was served
	last time.Time

	// the amount of time the limiter will sleep for
	// before serving the next request.
	sleep time.Duration

	// the minimum allowed time-per-reqeust (1sec/rps)
	perReq time.Duration

	// the maximum sleep time.
	// this is set in order to prevent a stampeding herd
	// of requests after a series of potentiatlly long-lived requests.
	maxSleep time.Duration
	clock    Clock
}

func New(rate int) *limiter {

	perReq := time.Second / time.Duration(rate)
	l := &limiter{
		perReq: perReq,

		maxSleep: -25 * perReq,
		clock:    clock.New(),
	}

	return l
}

func (l *limiter) WithClock(c clock.Clock) *limiter {
	l.clock = c
	return l
}

func (l *limiter) Take() time.Time {
	l.Lock()
	defer l.Unlock()

	now := l.clock.Now()

	if l.last.IsZero() {
		l.last = now
		return now
	}

	// amount of time needed to sleep in order to fulful the next request
	l.sleep += l.perReq - now.Sub(l.last)
	// maintain the ratelimiter's floor
	if l.sleep <= l.maxSleep {
		l.sleep = l.maxSleep
	}

	if l.sleep > 0 {
		// sleep and avdance the clock
		l.clock.Sleep(l.sleep)
		l.last = now.Add(l.sleep)
		l.sleep = 0
	} else {
		l.last = now
	}

	return l.last
}
