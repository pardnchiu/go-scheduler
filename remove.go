package cronJob

import (
	"container/heap"
)

func (c *cron) Remove(id int) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.running {
		c.remove <- id
		return
	}

	for i, entry := range c.heap {
		if entry.id == id {
			entry.enable = false
			heap.Remove(&c.heap, i)
			break
		}
	}
}
