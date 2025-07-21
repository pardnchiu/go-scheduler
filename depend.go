package goCron

import (
	"context"
	"fmt"
	"runtime"
	"time"
)

func newDepend() *depend {
	cpu := runtime.NumCPU()
	if cpu > 2 {
		maxWorker = cpu
	}

	return &depend{
		manager:  newDependManager(),
		stopChan: make(chan struct{}),
		queue:    make(chan Wait, 1024),
	}
}

func (d *depend) start() {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.running {
		return
	}
	d.running = true

	for i := 0; i < maxWorker; i++ {
		d.wait.Add(1)
		go d.worker()
	}
}

func (d *depend) stop() {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if !d.running {
		return
	}
	d.running = false

	close(d.stopChan)
	d.wait.Wait()
}

func (d *depend) worker() {
	defer d.wait.Done()

	for {
		select {
		case taskID := <-d.queue:
			d.runAfter(taskID)
		case <-d.stopChan:
			return
		}
	}
}

func (d *depend) addWait(id int64, delay time.Duration, state WaitState) {
	var timeout = 1 * time.Minute
	if delay > 0 {
		timeout = delay
	}
	d.queue <- Wait{
		ID:    id,
		Delay: timeout,
		State: state,
	}
}

// * Worker 執行的排序（v0.4.0 對 Worker 數進行了限制）
func (d *depend) runAfter(queue Wait) {
	task, isExist := d.manager.list[queue.ID]
	if !isExist {
		d.logger.Error(
			"Task not found",
			"ID", int(queue.ID),
		)
		return
	}

	task.mutex.RLock()
	status := task.state
	task.mutex.RUnlock()

	if status == TaskRunning || status == TaskCompleted {
		return
	}

	if err := d.manager.wait(queue.ID, queue.Delay); err != nil {
		result := taskResult{
			ID:     queue.ID,
			status: TaskFailed,
			start:  time.Now(),
			end:    time.Now(),
			error:  err,
		}
		d.manager.update(result)
		d.logger.Error(
			"Dependence Task failed",
			"ID", int(queue.ID),
			"error", err,
		)
		return
	}

	d.run(task)
}

func (d *depend) run(task *task) {
	start := time.Now()

	task.mutex.Lock()
	task.state = TaskRunning
	task.mutex.Unlock()

	d.logger.Info(
		"Task started",
		"ID", int(task.ID),
		"description", task.description,
	)

	var taskError error

	func() {
		defer func() {
			if r := recover(); r != nil {
				taskError = fmt.Errorf("task panic: %v", r)
				d.logger.Error(
					"Task panic",
					"ID", int(task.ID),
					"panic", r,
				)
			}
		}()

		if task.delay > 0 {
			ctx, cancel := context.WithTimeout(context.Background(), task.delay)
			defer cancel()

			done := make(chan error, 1)
			go func() {
				done <- task.action()
			}()

			select {
			case err := <-done:
				taskError = err
			case <-ctx.Done():
				taskError = fmt.Errorf("task timeout %d", task.delay)
				if task.onDelay != nil {
					task.onDelay()
				}
			}
		} else {
			taskError = task.action()
		}
	}()

	end := time.Now()
	duration := end.Sub(start)

	status := TaskCompleted
	if taskError != nil {
		status = TaskFailed
	}

	result := taskResult{
		ID:       task.ID,
		status:   status,
		start:    start,
		end:      end,
		duration: duration,
		error:    taskError,
	}

	d.manager.update(result)

	if taskError != nil {
		d.logger.Error(
			"Task failed",
			"ID", int(task.ID),
			"duration", duration,
			"error", taskError,
		)
	} else {
		d.logger.Info(
			"Task completed",
			"ID", int(task.ID),
			"duration", duration,
		)
	}
}
