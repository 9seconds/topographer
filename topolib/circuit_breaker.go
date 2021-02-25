package topolib

import (
	"context"
	"net/http"
	"sync/atomic"
	"time"
)

type circuitBreakerCallback func() (*http.Response, error)

const (
	circuitBreakerStateClosed uint32 = iota
	circuitBreakerStateHalfOpened
	circuitBreakerStateOpened
)

type circuitBreaker struct {
	state          uint32
	stateMutexChan chan bool

	halfOpenTimer        *time.Timer
	failuresCleanupTimer *time.Timer

	halfOpenAttempts uint32
	failuresCount    uint32

	openThreshold        uint32
	halfOpenTimeout      time.Duration
	resetFailuresTimeout time.Duration
}

func (c *circuitBreaker) Do(ctx context.Context, callback circuitBreakerCallback) (*http.Response, error) {
	switch atomic.LoadUint32(&c.state) {
	case circuitBreakerStateClosed:
		return c.doClosed(ctx, callback)
	case circuitBreakerStateHalfOpened:
		return c.doHalfOpened(ctx, callback)
	default:
		return nil, ErrCircuitBreakerOpened
	}
}

func (c *circuitBreaker) doClosed(ctx context.Context, callback circuitBreakerCallback) (*http.Response, error) {
	resp, err := callback()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case c.stateMutexChan <- true:
		defer func() {
			<-c.stateMutexChan
		}()
	}

	if err == nil {
		c.switchState(circuitBreakerStateClosed)

		return resp, err
	}

	c.failuresCount++

	if c.state == circuitBreakerStateClosed && c.failuresCount > c.openThreshold {
		c.switchState(circuitBreakerStateOpened)
	}

	return resp, err
}

func (c *circuitBreaker) doHalfOpened(ctx context.Context, callback circuitBreakerCallback) (*http.Response, error) {
	if !atomic.CompareAndSwapUint32(&c.halfOpenAttempts, 0, 1) {
		return nil, ErrCircuitBreakerOpened
	}

	resp, err := callback()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case c.stateMutexChan <- true:
		defer func() {
			<-c.stateMutexChan
		}()
	}

	if c.state != circuitBreakerStateHalfOpened {
		return resp, err
	}

	if err != nil {
		c.switchState(circuitBreakerStateOpened)
	} else {
		c.switchState(circuitBreakerStateClosed)
	}

	return resp, err
}

func (c *circuitBreaker) switchState(state uint32) {
	switch state {
	case circuitBreakerStateClosed:
		c.stopTimer(&c.halfOpenTimer)
		c.ensureTimer(&c.failuresCleanupTimer, c.resetFailuresTimeout, c.resetFailures)
	case circuitBreakerStateHalfOpened:
		c.stopTimer(&c.failuresCleanupTimer)
		c.stopTimer(&c.halfOpenTimer)
	case circuitBreakerStateOpened:
		c.stopTimer(&c.failuresCleanupTimer)
		c.ensureTimer(&c.halfOpenTimer, c.halfOpenTimeout, c.tryHalfOpen)
	}

	c.failuresCount = 0

	atomic.StoreUint32(&c.halfOpenAttempts, 0)
	atomic.StoreUint32(&c.state, state)
}

func (c *circuitBreaker) resetFailures() {
	c.stateMutexChan <- true

	defer func() {
		<-c.stateMutexChan
	}()

	c.stopTimer(&c.failuresCleanupTimer)

	if c.state == circuitBreakerStateClosed {
		c.switchState(circuitBreakerStateClosed)
	}
}

func (c *circuitBreaker) tryHalfOpen() {
	c.stateMutexChan <- true

	defer func() {
		<-c.stateMutexChan
	}()

	if c.state == circuitBreakerStateOpened {
		c.switchState(circuitBreakerStateHalfOpened)
	}
}

func (c *circuitBreaker) stopTimer(timerRef **time.Timer) {
	timer := *timerRef

	if timer == nil {
		return
	}

	timer.Stop()

	select {
	case <-timer.C:
	default:
	}

	*timerRef = nil
}

func (c *circuitBreaker) ensureTimer(timerRef **time.Timer, timeout time.Duration, callback func()) {
	if *timerRef == nil {
		*timerRef = time.AfterFunc(timeout, callback)
	}
}

func newCircuitBreaker(openThreshold uint32,
	halfOpenTimeout, resetFailuresTimeout time.Duration) *circuitBreaker {
	cb := &circuitBreaker{
		stateMutexChan:       make(chan bool, 1),
		openThreshold:        openThreshold,
		halfOpenTimeout:      halfOpenTimeout,
		resetFailuresTimeout: resetFailuresTimeout,
	}

	cb.switchState(circuitBreakerStateClosed)

	return cb
}
