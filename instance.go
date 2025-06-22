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

	cron.mutex.Lock()
	defer cron.mutex.Unlock()

	if !cron.running {
		cron.running = true

		go func() {
			now := time.Now().In(cron.location)

			for _, entry := range cron.heap {
				entry.next = entry.schedule.next(now)
			}
			heap.Init(&cron.heap)

			for {
				var timer *time.Timer
				var timerC <-chan time.Time

				if len(cron.heap) == 0 || cron.heap[0].next.IsZero() {
					timerC = nil
				} else {
					timer = time.NewTimer(cron.heap[0].next.Sub(now))
					timerC = timer.C
				}

				for {
					select {
					case now = <-timerC:
						now = now.In(cron.location)

						for len(cron.heap) > 0 && (cron.heap[0].next.Before(now) || cron.heap[0].next.Equal(now)) {
							e := heap.Pop(&cron.heap).(*task)

							if !e.enable {
								continue
							}

							cron.wait.Add(1)
							go func() {
								defer func() {
									if r := recover(); r != nil {
										fmt.Printf("Task panic recovered: %v\n", r)
									}
								}()
								defer cron.wait.Done()
								e.action()
							}()

							e.prev = e.next
							e.next = e.schedule.next(now)
							if !e.next.IsZero() {
								heap.Push(&cron.heap, e)
							}
						}

					case newEntry := <-cron.add:
						if timer != nil {
							timer.Stop()
						}
						now = time.Now().In(cron.location)
						newEntry.next = newEntry.schedule.next(now)
						heap.Push(&cron.heap, newEntry)

					case id := <-cron.remove:
						if timer != nil {
							timer.Stop()
						}
						now = time.Now().In(cron.location)
						cron.Remove(id)

					case <-cron.stop:
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

	return cron, nil
}

func (cron *cron) Stop() context.Context {
	cron.mutex.Lock()
	defer cron.mutex.Unlock()

	if cron.running {
		cron.stop <- struct{}{}
		cron.running = false
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		cron.wait.Wait()
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
