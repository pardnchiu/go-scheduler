package cronJob

import (
	"container/heap"
)

func (c *cron) Add(spec string, action func()) (int, error) {
	schedule, err := c.parser.parse(spec)
	if err != nil {
		return 0, c.logger.Error(err, "Failed to parse time spec")
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.next++
	entry := &task{
		id:       c.next,
		schedule: schedule,
		action:   c.chain.then(action),
		enable:   true,
	}

	if c.running {
		c.add <- entry
	} else {
		c.heap = append(c.heap, entry)
		heap.Init(&c.heap)
	}

	return entry.id, nil
}

func (t taskChain) then(a func()) func() {
	for i := range t {
		a = t[len(t)-i-1](a)
	}
	return a
}
