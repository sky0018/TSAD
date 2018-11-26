package tsfetcher

import (
	"sync/atomic"
	"time"
)

// ConcLimiter .
type ConcLimiter struct {
	cnt int64
	max int64
}

// NewConcLimiter .
func NewConcLimiter(max int64) *ConcLimiter {
	return &ConcLimiter{
		cnt: 0,
		max: max,
	}
}

// Wait .
func (cl *ConcLimiter) Wait() {
	for {
		if cl.Take() {
			return
		}
		time.Sleep(time.Millisecond * 2)
	}
}

// Take .
func (cl *ConcLimiter) Take() bool {
	newCnt := atomic.AddInt64(&cl.cnt, 1)
	if newCnt <= cl.max {
		return true
	}
	atomic.AddInt64(&cl.cnt, -1)
	return false
}

// Release .
func (cl *ConcLimiter) Release() {
	atomic.AddInt64(&cl.cnt, -1)
}
