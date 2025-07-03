package cron

import (
	"container/heap"
	"context"
	"fmt"
	"time"

	goLogger "github.com/pardnchiu/go-logger"
)

func New(c Config) (*cron, error) {
	c.Log = validLoggerConfig(c)

	location := time.Local
	if c.Location != nil {
		location = c.Location
	}

	logger, err := goLogger.New(c.Log)
	if err != nil {
		return nil, fmt.Errorf("Failed to initialize `pardnchiu/go-logger`: %w", err)
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
							go func() {
								defer func() {
									if r := recover(); r != nil {
										c.logger.Info("Recovered from panic [task: %d]", e.ID)
									}
								}()
								defer c.wait.Done()
								e.Action()
							}()

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

func validLoggerConfig(c Config) *Log {
	if c.Log == nil {
		c.Log = &Log{
			Path:    defaultLogPath,
			Stdout:  false,
			MaxSize: defaultLogMaxSize,
		}
	}
	if c.Log.Path == "" {
		c.Log.Path = defaultLogPath
	}
	if c.Log.MaxSize <= 0 {
		c.Log.MaxSize = defaultLogMaxSize
	}
	if c.Log.MaxBackup <= 0 {
		c.Log.MaxBackup = defaultLogMaxBackup
	}
	return c.Log
}
