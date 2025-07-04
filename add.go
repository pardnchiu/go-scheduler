package cron

import (
	"container/heap"
	"sync/atomic"
)

func (c *cron) Add(spec string, action func(), description ...string) (int64, error) {
	schedule, err := c.parser.parse(spec)
	if err != nil {
		c.logger.Error("Failed to parse time spec", err)
		return 0, err
	}

	c.logger.Info("adding task [spec: %s, description: %s]", spec, description)

	c.mutex.Lock()
	defer c.mutex.Unlock()

	entry := &task{
		ID:       atomic.AddInt64(&c.next, 1),
		Schedule: schedule,
		Action:   c.chain.then(action),
		Enable:   true,
	}

	if len(description) > 0 {
		entry.Description = description[0]
	}
	c.logger.Debug("task created [ID: %d, Description: %s]", entry.ID, entry.Description)

	if c.running {
		c.add <- entry
	} else {
		c.heap = append(c.heap, entry)
		heap.Init(&c.heap)
		c.logger.Debug("task added to heap [ID: %d]", entry.ID)
	}

	return entry.ID, nil
}

func (t taskChain) then(a func()) func() {
	for i := range t {
		a = t[len(t)-i-1](a)
	}
	return a
}
