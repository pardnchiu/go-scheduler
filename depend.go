package goCron

import (
	"context"
	"fmt"
	"log/slog"
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
		queue:    make(chan int64),
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

func (d *depend) add(id int64) {
	d.queue <- id
}

// * Worker 執行的排序（v0.4.0 對 Worker 數進行了限制）
func (d *depend) runAfter(id int64) {
	task, isExist := d.manager.list[id]
	if !isExist {
		slog.Error(
			"Task not found",
			slog.Int("ID", int(id)),
		)
		return
	}

	task.mutex.RLock()
	status := task.state
	task.mutex.RUnlock()

	if status == TaskRunning || status == TaskCompleted {
		return
	}

	// TODO: 後續改為用戶自己決定超時時間
	// * 設置超時等待（預設 1 分鐘）
	if err := d.manager.wait(id, 1*time.Minute); err != nil {
		result := taskResult{
			ID:     id,
			status: TaskFailed,
			start:  time.Now(),
			end:    time.Now(),
			error:  err,
		}
		d.manager.update(result)
		slog.Error(
			"Dependence Task failed",
			slog.Int("ID", int(id)),
			slog.Any("error", err),
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

	slog.Info(
		"Task started",
		slog.Int("ID", int(task.ID)),
		slog.String("description", task.description),
	)

	var taskError error

	func() {
		defer func() {
			if r := recover(); r != nil {
				taskError = fmt.Errorf("task panic: %v", r)
				slog.Error(
					"Task panic",
					slog.Int("ID", int(task.ID)),
					slog.Any("panic", r),
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
				taskError = fmt.Errorf("Task timeout %d", task.delay)
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
		slog.Error(
			"Task failed",
			slog.Int("ID", int(task.ID)),
			slog.Duration("duration", duration),
			slog.Any("error", taskError),
		)
	} else {
		slog.Info(
			"Task completed",
			slog.Int("ID", int(task.ID)),
			slog.Duration("duration", duration),
		)
	}
}
