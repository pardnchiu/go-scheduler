package goCron

import (
	"container/heap"
	"fmt"
	"sync/atomic"
	"time"
)

func (c *cron) Add(spec string, action interface{}, arg ...interface{}) (int64, error) {
	schedule, err := c.parser.parse(spec)
	if err != nil {
		return 0, fmt.Errorf("Failed to parse: %w", err)
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	entry := &task{
		ID:       atomic.AddInt64(&c.next, 1),
		schedule: schedule,
		enable:   true,
		state:    TaskPending,
	}

	withError := false

	switch v := action.(type) {
	// * 無返回錯誤值
	case func():
		entry.action = func() error {
			v()
			return nil
		}
		// * 設為已完成
		entry.state = TaskCompleted
	// * 有返回錯誤值
	case func() error:
		// * 標記有回傳值
		withError = true
		entry.action = v
	default:
		return 0, fmt.Errorf("Action need to be func() or func()")
	}

	var after []int64
	for _, e := range arg {
		switch v := e.(type) {
		case string:
			entry.description = v
		case time.Duration:
			entry.delay = v
		case func():
			entry.onDelay = v
		// * 依賴任務
		case []int64:
			after = v
			entry.state = TaskPending
		}
	}

	if !withError && after != nil {
		return 0, fmt.Errorf("Need return value to get dependence support")
	}

	if after != nil {
		entry.after = make([]int64, len(after))
		copy(entry.after, after)
	}

	if c.running {
		c.add <- entry
	} else {
		c.heap = append(c.heap, entry)
		heap.Init(&c.heap)
		c.depend.manager.add(entry)
	}

	return entry.ID, nil
}
