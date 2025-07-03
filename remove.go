package cron

import (
	"container/heap"
)

func (c *cron) RemoveAll() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.running {
		c.remove <- 0
		return
	}

	for i := range c.heap {
		c.heap[i].Enable = false
	}
	heap.Init(&c.heap)
}

func (c *cron) Remove(id int) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.running {
		c.remove <- id
		return
	}

	for i, entry := range c.heap {
		if entry.ID == id {
			entry.Enable = false
			heap.Remove(&c.heap, i)
			break
		}
	}
}
