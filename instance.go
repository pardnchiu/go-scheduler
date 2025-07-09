package goCron

import (
	"container/heap"
	"context"
	"log/slog"
	"time"
)

func New(c Config) (*cron, error) {
	location := time.Local
	if c.Location != nil {
		location = c.Location
	}

	cron := &cron{
		heap:      make(taskHeap, 0),
		chain:     taskChain{},
		parser:    parser{},
		stop:      make(chan struct{}),
		add:       make(chan *task),
		remove:    make(chan int64),
		removeAll: make(chan struct{}),
		location:  location,
		running:   false,
	}

	return cron, nil
}

func (c *cron) Start() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.running {
		c.running = true

		go func() {
			now := time.Now().In(c.location)

			for _, entry := range c.heap {
				entry.Next = entry.Schedule.next(now)
			}
			heap.Init(&c.heap)

			for {
				var timer *time.Timer
				var timerC <-chan time.Time

				if len(c.heap) == 0 || c.heap[0].Next.IsZero() {
					timerC = nil
				} else {
					timer = time.NewTimer(c.heap[0].Next.Sub(now))
					timerC = timer.C
				}

				for {
					select {
					case now = <-timerC:
						now = now.In(c.location)

						for len(c.heap) > 0 && (c.heap[0].Next.Before(now) || c.heap[0].Next.Equal(now)) {
							e := heap.Pop(&c.heap).(*task)

							if !e.Enable {
								continue
							}

							c.wait.Add(1)
							go func(entry *task) {
								defer func() {
									if r := recover(); r != nil {
										slog.Info("Recovered from panic", slog.Int("taskID", int(entry.ID)), slog.Any("error", r))
									}
								}()
								defer c.wait.Done()

								if entry.Delay > 0 {
									ctx, cancel := context.WithTimeout(context.Background(), entry.Delay)
									defer cancel()

									done := make(chan struct{})
									go func() {
										defer cancel()
										entry.Action()
										close(done)
									}()

									select {
									case <-done:
										// 任務正常完成，不觸發超時
									case <-ctx.Done():
										// 任務超時，觸發超時
										if entry.OnDelay != nil {
											entry.OnDelay()
										}
										slog.Info("Task timeout", slog.Int("taskID", int(entry.ID)), slog.Duration("delay", entry.Delay))
									}
								} else {
									entry.Action()
								}
							}(e)

							e.Prev = e.Next
							e.Next = e.Schedule.next(now)
							if !e.Next.IsZero() {
								heap.Push(&c.heap, e)
							}
						}

					case newEntry := <-c.add:
						if timer != nil {
							timer.Stop()
						}
						now = time.Now().In(c.location)
						newEntry.Next = newEntry.Schedule.next(now)
						heap.Push(&c.heap, newEntry)

					case id := <-c.remove:
						if timer != nil {
							timer.Stop()
						}
						now = time.Now().In(c.location)
						for i, entry := range c.heap {
							if entry.ID == id {
								entry.Enable = false
								heap.Remove(&c.heap, i)
								break
							}
						}

					case <-c.removeAll:
						if timer != nil {
							timer.Stop()
						}
						now = time.Now().In(c.location)
						// 完全清空 heap
						for len(c.heap) > 0 {
							heap.Pop(&c.heap)
						}

					case <-c.stop:
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
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		c.wait.Wait()
		cancel()
	}()

	return ctx
}
