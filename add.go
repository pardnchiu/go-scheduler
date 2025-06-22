package cronJob

import (
	"container/heap"
)

func (cron *cron) Add(spec string, action func()) (int, error) {
	schedule, err := cron.parser.parse(spec)
	if err != nil {
		return 0, cron.logger.Error(err, "Failed to parse time spec")
	}

	cron.mutex.Lock()
	defer cron.mutex.Unlock()

	cron.next++
	entry := &task{
		id:       cron.next,
		schedule: schedule,
		action:   cron.chain.then(action),
		enable:   true,
	}

	if !cron.running {
		cron.heap = append(cron.heap, entry)
		heap.Init(&cron.heap)
	} else {
		cron.add <- entry
	}

	return entry.id, nil
}

func (t taskChain) then(a func()) func() {
	for i := range t {
		a = t[len(t)-i-1](a)
	}
	return a
}
