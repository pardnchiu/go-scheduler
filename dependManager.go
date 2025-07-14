package goCron

import (
	"context"
	"fmt"
	"time"
)

func newDependManager() *dependManager {
	return &dependManager{
		list:    make(map[int64]*task),
		waiting: make(map[int64][]*task),
	}
}

func (m *dependManager) add(t *task) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.list[t.ID] = t

	t.mutex.RLock()
	hasAfter := len(t.after) > 0
	t.mutex.RUnlock()

	// * 存在依賴任務
	if hasAfter {
		t.startChan = make(chan struct{}, 1)
		t.doneChan = make(chan taskResult, 1)
	}

	if t.state == 0 {
		t.state = TaskPending
	}
}

func (m *dependManager) check(id int64) taskState {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	task, isExist := m.list[id]
	// * 任務不存在
	if !isExist {
		return taskState{
			done:  false,
			error: fmt.Errorf("Task not found: %d", id),
		}
	}

	task.mutex.RLock()
	defer task.mutex.RUnlock()

	var waiting []int64

	for _, afterId := range task.after {
		afterTask, isExist := m.list[afterId]
		// * 依賴任務不存在
		if !isExist {
			return taskState{
				done:   false,
				failed: &afterId,
				error:  fmt.Errorf("Dependence Task not found: %d", id),
			}
		}

		afterTask.mutex.RLock()
		status := afterTask.state
		afterTask.mutex.RUnlock()

		// * 依賴任務執行錯誤
		if status == TaskFailed {
			return taskState{
				done:   false,
				failed: &afterId,
				error:  fmt.Errorf("Dependence Task is failed: %d", id),
			}
		}

		// * 依賴任務未完成
		if status != TaskCompleted {
			waiting = append(waiting, afterId)
		}
	}

	// * 尚有依賴任務未完成
	if len(waiting) > 0 {
		return taskState{
			done:    false,
			waiting: waiting,
			error:   fmt.Errorf("Waiting for dependencies: %d", waiting),
		}
	}

	return taskState{
		done: true,
	}
}

// TODO: 後續改寫觸發的方式
func (m *dependManager) wait(id int64, timeout time.Duration) error {
	// * context 超時控制
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for {
		result := m.check(id)
		// * 依賴任務接完成
		if result.done {
			return nil
		}

		// * 依賴任務失敗
		if result.failed != nil {
			return fmt.Errorf("Dependence Task failed: %d, %s", *result.failed, result.error.Error())
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("Timeout waiting for dependencies: %s", result.error.Error())
		case <-time.After(1 * time.Millisecond):
		}
	}
}

func (m *dependManager) update(result taskResult) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if task, isExist := m.list[result.ID]; isExist {
		task.mutex.Lock()
		task.state = result.status
		task.result = &result
		task.mutex.Unlock()

		// * 完成通知
		select {
		case task.doneChan <- result:
		default:
		}
	}
}
