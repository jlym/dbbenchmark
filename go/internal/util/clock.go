package util

import (
	"sync"
	"time"
)

type Clock interface {
	NowUtc() time.Time
}

type RealClock struct{}

func NewRealClock() *RealClock {
	return &RealClock{}
}

func (c *RealClock) NowUtc() time.Time {
	return time.Now().UTC()
}

type StubClock struct {
	now  time.Time
	lock sync.Mutex
}

func NewStubClock() *StubClock {
	clock := &StubClock{}
	clock.UpdateNow()
	return clock
}

func (c *StubClock) NowUtc() time.Time {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.now
}

func (c *StubClock) SetNow(now time.Time) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.now = now.UTC()
}

func (c *StubClock) UpdateNow() time.Time {
	now := time.Now().UTC()
	c.SetNow(now)
	return now
}
