package goCron

import (
	"container/heap"
	"fmt"
	"sync/atomic"
	"time"
)

func (c *cron) Add(spec string, action func(), args ...interface{}) (int64, error) {
	schedule, err := c.parser.parse(spec)
	if err != nil {
		return 0, fmt.Errorf("Failed to parse schedule spec: %w", err)
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	entry := &task{
		ID:       atomic.AddInt64(&c.next, 1),
		Schedule: schedule,
		Action:   c.chain.then(action),
		Enable:   true,
	}

	for _, arg := range args {
		switch v := arg.(type) {
		case string:
			entry.Description = v
		case time.Duration:
			entry.Delay = v
		case func():
			entry.OnDelay = v
		}
	}

	if c.running {
		c.add <- entry
	} else {
		c.heap = append(c.heap, entry)
		heap.Init(&c.heap)
	}

	return entry.ID, nil
}

func (t taskChain) then(a func()) func() {
	for i := range t {
		a = t[len(t)-i-1](a)
	}
	return a
}
