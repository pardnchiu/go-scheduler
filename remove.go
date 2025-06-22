package cronJob

import (
	"container/heap"
)

func (cron *cron) Remove(id int) {
	cron.mutex.Lock()
	defer cron.mutex.Unlock()

	if cron.running {
		cron.remove <- id
		return
	}

	for i, entry := range cron.heap {
		if entry.id == id {
			entry.enable = false
			heap.Remove(&cron.heap, i)
			break
		}
	}
}
