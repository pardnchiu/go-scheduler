package cron

import (
	"container/heap"
)

func (c *cron) RemoveAll() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.running {
		c.removeAll <- struct{}{}
		return
	}

	c.logger.Info("removing all tasks")

	for i := range c.heap {
		c.heap[i].Enable = false
	}
	heap.Init(&c.heap)
}

func (c *cron) Remove(id int64) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.logger.Info("removing task [ID: %d]", id)

	if c.running {
		c.remove <- id
		return
	}

	for i, entry := range c.heap {
		if entry.ID == id {
			entry.Enable = false
			heap.Remove(&c.heap, i)
			c.logger.Info("removed task [ID: %d, Description: %s]", entry.ID, entry.Description)
			break
		}
	}
}
