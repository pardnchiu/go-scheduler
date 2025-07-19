package goCron

import (
	"container/heap"
	"context"
	"fmt"
	"log/slog"
	"log/syslog"
	"os"
	"time"
)

func New(c Config) (*cron, error) {
	location := time.Local
	if c.Location != nil {
		location = c.Location
	}

	writer, err := syslog.New(syslog.LOG_INFO|syslog.LOG_LOCAL0, "goCron")
	if err != nil {
		logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
	} else {
		logger = slog.New(slog.NewJSONHandler(writer, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
	}

	cron := &cron{
		heap:      make(taskHeap, 0),
		parser:    parser{},
		stop:      make(chan struct{}),
		add:       make(chan *task),
		remove:    make(chan int64),
		removeAll: make(chan struct{}),
		location:  location,
		running:   false,
		depend:    newDepend(),
	}

	return cron, nil
}

func (c *cron) Start() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.running {
		c.running = true
		c.depend.start()

		go func() {
			now := time.Now().In(c.location)

			for _, entry := range c.heap {
				entry.next = entry.schedule.next(now)
			}
			heap.Init(&c.heap)

			for {
				var timer *time.Timer
				var timerC <-chan time.Time

				if len(c.heap) == 0 || c.heap[0].next.IsZero() {
					timerC = nil
				} else {
					timer = time.NewTimer(c.heap[0].next.Sub(now))
					timerC = timer.C
				}

				for {
					select {
					case now = <-timerC:
						// * 時間觸發
						now = now.In(c.location)

						for len(c.heap) > 0 && (c.heap[0].next.Before(now) || c.heap[0].next.Equal(now)) {
							e := heap.Pop(&c.heap).(*task)

							if !e.enable {
								continue
							}

							c.run(e)

							e.prev = e.next
							e.next = e.schedule.next(now)
							if !e.next.IsZero() {
								heap.Push(&c.heap, e)
							}
						}

					case newEntry := <-c.add:
						// * 新增任務觸發
						if timer != nil {
							timer.Stop()
						}
						now = time.Now().In(c.location)
						newEntry.next = newEntry.schedule.next(now)
						heap.Push(&c.heap, newEntry)
						c.depend.manager.add(newEntry)

					case id := <-c.remove:
						// * 移除任務觸發
						if timer != nil {
							timer.Stop()
						}
						now = time.Now().In(c.location)
						for i, entry := range c.heap {
							if entry.ID == id {
								entry.enable = false
								heap.Remove(&c.heap, i)
								break
							}
						}

					case <-c.removeAll:
						// * 移除任務觸發
						if timer != nil {
							timer.Stop()
						}
						now = time.Now().In(c.location)
						// 完全清空 heap
						for len(c.heap) > 0 {
							heap.Pop(&c.heap)
						}

					case <-c.stop:
						// * 移除任務觸發
						if timer != nil {
							timer.Stop()
						}
						return
					}
					break
				}
			}
		}()
	}
}

func (c *cron) Stop() context.Context {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.running {
		c.stop <- struct{}{}
		c.running = false
		c.depend.stop()
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		c.wait.Wait()
		cancel()
	}()

	return ctx
}

func (c *cron) run(e *task) {
	e.mutex.RLock()
	hasDeps := len(e.after) > 0
	e.mutex.RUnlock()

	if hasDeps {
		c.depend.add(e.ID)
	} else {
		c.runAfter(e)
	}
}

func (c *cron) runAfter(e *task) {
	c.wait.Add(1)
	go func(entry *task) {
		defer func() {
			if r := recover(); r != nil {
				// * 更新狀態至錯誤
				entry.mutex.Lock()
				entry.state = TaskFailed
				entry.mutex.Unlock()

				logger.Info(
					"Recovered from panic",
					"ID", int(entry.ID),
					"error", r,
				)
			}
		}()
		defer c.wait.Done()

		// * 更新狀態至執行中
		entry.mutex.Lock()
		entry.state = TaskRunning
		entry.mutex.Unlock()

		var taskError error
		if entry.delay > 0 {
			ctx, cancel := context.WithTimeout(context.Background(), entry.delay)
			defer cancel()

			done := make(chan struct{})
			go func() {
				defer cancel()

				if err := entry.action(); err != nil {
					taskError = err
					logger.Error(
						"Task failed",
						"error", err,
					)
				}
				close(done)
			}()

			select {
			case <-done:
			case <-ctx.Done():
				// * 任務超時
				taskError = fmt.Errorf("Task timeout: %d", entry.delay)
				if entry.onDelay != nil {
					entry.onDelay()
				}
				logger.Warn(
					"Task timeout",
					"ID", int(entry.ID),
					"delay", entry.delay,
				)
			}
		} else {
			if err := entry.action(); err != nil {
				taskError = err
				logger.Error(
					"Task failed",
					"error", err,
				)
			}
		}

		entry.mutex.Lock()
		if taskError != nil {
			// * 更新狀態至錯誤
			entry.state = TaskFailed
		} else {
			// * 更新狀態至完成
			entry.state = TaskCompleted
		}
		entry.mutex.Unlock()
	}(e)
}
