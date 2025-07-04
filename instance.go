package cron

import (
	"container/heap"
	"context"
	"time"
)

func New(c Config) (*cron, error) {
	logger := c.Logger
	if logger == nil {
		logger = NewLogger()
	}

	location := time.Local
	if c.Location != nil {
		location = c.Location
	}

	cron := &cron{
		logger:    logger,
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

	logger.Info("cron instance created")
	return cron, nil
}

func (c *cron) Start() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.running {
		c.running = true
		c.logger.Info("starting cron scheduler")

		go func() {
			now := time.Now().In(c.location)

			for _, entry := range c.heap {
				entry.Next = entry.Schedule.next(now)
			}
			heap.Init(&c.heap)

			c.logger.Debug("cron scheduler initialized with %d tasks", len(c.heap))

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

							c.logger.Debug("executing task [ID: %d, Description: %s]", e.ID, e.Description)

							c.wait.Add(1)
							go func(task *task) {
								defer func() {
									if r := recover(); r != nil {
										c.logger.Error("recovered from panic [task: %d, Description: %s]: %v", task.ID, task.Description, r)
									}
								}()
								defer c.wait.Done()
								task.Action()
							}(e)

							e.Prev = e.Next
							e.Next = e.Schedule.next(now)
							if !e.Next.IsZero() {
								heap.Push(&c.heap, e)
								c.logger.Debug("rescheduled task [ID: %d] for next execution at %v", e.ID, e.Next)
							} else {
								c.logger.Debug("task [ID: %d] completed (no next execution)", e.ID)
							}
						}

					case newEntry := <-c.add:
						if timer != nil {
							timer.Stop()
						}
						now = time.Now().In(c.location)
						newEntry.Next = newEntry.Schedule.next(now)
						heap.Push(&c.heap, newEntry)
						c.logger.Info("added task [ID: %d, Description: %s], next execution: %v", newEntry.ID, newEntry.Description, newEntry.Next)

					case id := <-c.remove:
						if timer != nil {
							timer.Stop()
						}
						now = time.Now().In(c.location)
						removed := false
						for i, entry := range c.heap {
							if entry.ID == id {
								entry.Enable = false
								heap.Remove(&c.heap, i)
								c.logger.Info("removed task [ID: %d, Description: %s]", entry.ID, entry.Description)
								removed = true
								break
							}
						}
						if !removed {
							c.logger.Debug("attempted to remove non-existent task [ID: %d]", id)
						}

					case <-c.removeAll:
						if timer != nil {
							timer.Stop()
						}
						now = time.Now().In(c.location)
						count := len(c.heap)
						// 完全清空 heap
						for len(c.heap) > 0 {
							heap.Pop(&c.heap)
						}
						c.logger.Info("removed all tasks [count: %d]", count)

					case <-c.stop:
						if timer != nil {
							timer.Stop()
						}
						c.logger.Info("stopping cron scheduler")
						return
					}
					break
				}
			}
		}()
		c.logger.Info("cron scheduler started successfully")
	} else {
		c.logger.Debug("attempted to start already running scheduler")
	}
}

func (c *cron) Stop() context.Context {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.running {
		c.logger.Info("stopping cron scheduler...")
		c.stop <- struct{}{}
		c.running = false
	} else {
		c.logger.Debug("attempted to stop already stopped scheduler")
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		c.wait.Wait()
		c.logger.Info("all tasks completed, cron scheduler stopped")
		cancel()
	}()

	return ctx
}
