package cronJob

import (
	"container/heap"
	"context"
	"fmt"
	"time"

	goLogger "github.com/pardnchiu/go-logger"
)

func New(c Config) (*cron, error) {
	c.Log = validLoggerConfig(c)

	logger, err := goLogger.New(c.Log)
	if err != nil {
		return nil, fmt.Errorf("Failed to initialize `pardnchiu/go-logger`: %w", err)
	}

	cron := &cron{
		logger:   logger,
		heap:     make(taskHeap, 0),
		chain:    taskChain{},
		parser:   parser{},
		stop:     make(chan struct{}),
		add:      make(chan *task),
		remove:   make(chan int),
		location: time.Local,
		running:  false,
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
						now = now.In(c.location)

						for len(c.heap) > 0 && (c.heap[0].next.Before(now) || c.heap[0].next.Equal(now)) {
							e := heap.Pop(&c.heap).(*task)

							if !e.enable {
								continue
							}

							c.wait.Add(1)
							go func() {
								defer func() {
									if r := recover(); r != nil {
										fmt.Printf("Task panic recovered: %v\n", r)
									}
								}()
								defer c.wait.Done()
								e.action()
							}()

							e.prev = e.next
							e.next = e.schedule.next(now)
							if !e.next.IsZero() {
								heap.Push(&c.heap, e)
							}
						}

					case newEntry := <-c.add:
						if timer != nil {
							timer.Stop()
						}
						now = time.Now().In(c.location)
						newEntry.next = newEntry.schedule.next(now)
						heap.Push(&c.heap, newEntry)

					case id := <-c.remove:
						if timer != nil {
							timer.Stop()
						}
						now = time.Now().In(c.location)
						c.Remove(id)

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
