/*
 * unit_test.go
 * this file is created by github copilot
 * to test the cron package
 */

package main

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	goCron "github.com/pardnchiu/go-cron"
)

func TestCronCreation(t *testing.T) {
	config := goCron.Config{}
	c, err := goCron.New(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if c == nil {
		t.Fatal("Expected cron instance, got nil")
	}
}

func TestCronWithoutLog(t *testing.T) {
	config := goCron.Config{}
	c, err := goCron.New(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if c == nil {
		t.Fatal("Expected cron instance, got nil")
	}
}

func TestCronEveryThirtySeconds(t *testing.T) {
	config := goCron.Config{}
	c, err := goCron.New(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	var mu sync.Mutex
	count := 0

	_, err = c.Add("@every 30s", func() {
		mu.Lock()
		count++
		mu.Unlock()
	})

	if err != nil {
		t.Fatalf("Expected no error adding task, got %v", err)
	}

	c.Start()

	// 等待 35 秒確保任務至少執行一次
	time.Sleep(35 * time.Second)

	ctx := c.Stop()
	<-ctx.Done()

	mu.Lock()
	finalCount := count
	mu.Unlock()

	if finalCount < 1 {
		t.Fatalf("Expected count to be at least 1, got %d", finalCount)
	}
}

func TestCronStartStop(t *testing.T) {
	config := goCron.Config{}
	c, err := goCron.New(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	executed := false
	_, err = c.Add("@every 30s", func() {
		executed = true
		fmt.Print("Task executed\n", time.Now().Format("15:04:05"), "\n", executed)
	}, "Test Task")

	if err != nil {
		t.Fatalf("Expected no error adding task, got %v", err)
	}

	c.Start()

	time.Sleep(500 * time.Millisecond)

	ctx := c.Stop()

	select {
	case <-ctx.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("Stop context should complete within 2 seconds")
	}
}

func TestCronMultipleTasks(t *testing.T) {
	config := goCron.Config{}
	c, err := goCron.New(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	var mu sync.Mutex
	count1 := 0
	count2 := 0

	_, err = c.Add("@every 30s", func() {
		mu.Lock()
		count1++
		mu.Unlock()
	}, "Task 1")

	if err != nil {
		t.Fatalf("Expected no error adding task 1, got %v", err)
	}

	_, err = c.Add("@every 30s", func() {
		mu.Lock()
		count2++
		mu.Unlock()
	})

	if err != nil {
		t.Fatalf("Expected no error adding task 2, got %v", err)
	}

	c.Start()

	// 等待 35 秒確保任務至少執行一次
	time.Sleep(35 * time.Second)

	ctx := c.Stop()
	<-ctx.Done()

	mu.Lock()
	finalCount1 := count1
	finalCount2 := count2
	mu.Unlock()

	if finalCount1 < 1 {
		t.Fatalf("Expected count1 to be at least 1, got %d", finalCount1)
	}
	if finalCount2 < 1 {
		t.Fatalf("Expected count2 to be at least 1, got %d", finalCount2)
	}
}

func TestCronStopWithoutStart(t *testing.T) {
	config := goCron.Config{}
	c, err := goCron.New(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	ctx := c.Stop()

	select {
	case <-ctx.Done():
	case <-time.After(1 * time.Second):
		t.Fatal("Stop should complete immediately when not started")
	}
}

func TestCronTaskPanic(t *testing.T) {
	config := goCron.Config{}
	c, err := goCron.New(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	var mu sync.Mutex
	normalTaskExecuted := false

	_, err = c.Add("@every 30s", func() {
		panic("test panic")
	})

	if err != nil {
		t.Fatalf("Expected no error adding panic task, got %v", err)
	}

	_, err = c.Add("@every 30s", func() {
		mu.Lock()
		normalTaskExecuted = true
		mu.Unlock()
	})

	if err != nil {
		t.Fatalf("Expected no error adding normal task, got %v", err)
	}

	c.Start()

	// 等待 35 秒確保任務執行
	time.Sleep(35 * time.Second)

	ctx := c.Stop()
	<-ctx.Done()

	mu.Lock()
	executed := normalTaskExecuted
	mu.Unlock()

	if !executed {
		t.Fatal("Normal task should execute even when other task panics")
	}
}

func TestCronInvalidSchedule(t *testing.T) {
	c, _ := goCron.New(goCron.Config{})

	// 期望返回錯誤而不是 panic
	_, err := c.Add("invalid-cron-expression", func() {}, "Invalid Schedule Test")
	if err == nil {
		t.Error("Expected error for invalid schedule, but got none")
	}
}

func TestCronInvalidMinInterval(t *testing.T) {
	c, _ := goCron.New(goCron.Config{})

	// 測試小於 30 秒的間隔應該返回錯誤
	_, err := c.Add("@every 1s", func() {}, "Too Frequent Task")
	if err == nil {
		t.Error("Expected error for interval less than 30s, but got none")
	}

	_, err = c.Add("@every 29s", func() {}, "Too Frequent Task")
	if err == nil {
		t.Error("Expected error for interval less than 30s, but got none")
	}

	// 測試 30 秒應該成功
	_, err = c.Add("@every 30s", func() {}, "Valid Task")
	if err != nil {
		t.Errorf("Expected no error for 30s interval, got %v", err)
	}
}

func TestCronList(t *testing.T) {
	config := goCron.Config{}
	c, err := goCron.New(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// 新增多個任務
	id1, err := c.Add("@every 30s", func() {}, "Task 1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	id2, err := c.Add("@every 60s", func() {}, "Task 2")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// 取得任務清單
	tasks := c.List()

	// 驗證任務數量
	if len(tasks) != 2 {
		t.Fatalf("Expected 2 tasks, got %d", len(tasks))
	}

	// 驗證任務內容
	foundTask1 := false
	foundTask2 := false

	for _, task := range tasks {
		if task.ID == id1 {
			foundTask1 = true
		}
		if task.ID == id2 {
			foundTask2 = true
		}
	}

	if !foundTask1 {
		t.Fatal("Task 1 not found in list")
	}
	if !foundTask2 {
		t.Fatal("Task 2 not found in list")
	}
}

func TestCronListAfterRemove(t *testing.T) {
	config := goCron.Config{}
	c, err := goCron.New(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// 新增任務
	id1, err := c.Add("@every 30s", func() {}, "Task 1")
	if err != nil {
		t.Fatalf("Expected no error adding task 1, got %v", err)
	}

	id2, err := c.Add("@every 60s", func() {}, "Task 2")
	if err != nil {
		t.Fatalf("Expected no error adding task 2, got %v", err)
	}

	// 移除一個任務
	c.Remove(id1)

	// 取得任務清單
	tasks := c.List()

	// 驗證只剩一個任務
	if len(tasks) != 1 {
		t.Fatalf("Expected 1 task after removal, got %d", len(tasks))
	}

	// 驗證剩餘任務
	if tasks[0].ID != id2 {
		t.Fatal("Wrong task remained after removal")
	}
}

func TestCronListEmpty(t *testing.T) {
	config := goCron.Config{}
	c, err := goCron.New(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// 取得空的任務清單
	tasks := c.List()

	// 驗證清單為空
	if len(tasks) != 0 {
		t.Fatalf("Expected empty list, got %d tasks", len(tasks))
	}
}

func TestCronRemoveAll(t *testing.T) {
	config := goCron.Config{}
	c, err := goCron.New(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// 新增多個任務
	_, err = c.Add("@every 30s", func() {}, "Task 1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	_, err = c.Add("@every 60s", func() {}, "Task 2")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	_, err = c.Add("@every 90s", func() {}, "Task 3")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// 驗證任務已新增
	tasks := c.List()
	if len(tasks) != 3 {
		t.Fatalf("Expected 3 tasks, got %d", len(tasks))
	}

	// 移除所有任務
	c.RemoveAll()

	// 驗證所有任務已移除
	tasks = c.List()
	if len(tasks) != 0 {
		t.Fatalf("Expected 0 tasks after RemoveAll, got %d", len(tasks))
	}
}

func TestCronRemoveAllEmptyList(t *testing.T) {
	config := goCron.Config{}
	c, err := goCron.New(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// 在空清單上執行 RemoveAll
	c.RemoveAll()

	// 驗證清單仍為空
	tasks := c.List()
	if len(tasks) != 0 {
		t.Fatalf("Expected 0 tasks, got %d", len(tasks))
	}
}

func TestCronRemoveAllWithRunningTasks(t *testing.T) {
	config := goCron.Config{}
	c, err := goCron.New(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// 新增任務
	_, err = c.Add("@every 30s", func() {}, "Running Task")
	if err != nil {
		t.Fatalf("Expected no error adding task, got %v", err)
	}

	// 啟動 cron
	c.Start()

	// 等待調度器啟動
	time.Sleep(50 * time.Millisecond)

	// 移除所有任務
	c.RemoveAll()

	time.Sleep(50 * time.Millisecond)

	// 驗證任務已移除
	if len(c.List()) != 0 {
		t.Fatal("Expected 0 tasks after RemoveAll")
	}

	// 停止 cron
	ctx := c.Stop()
	select {
	case <-ctx.Done():
	case <-time.After(1 * time.Second):
		t.Fatal("Stop should complete within 1 second")
	}
}

func TestTaskTimeout(t *testing.T) {
	config := goCron.Config{}

	cron, err := goCron.New(config)
	if err != nil {
		t.Fatalf("建立 cron 失敗: %v", err)
	}

	var timeoutTriggered bool
	var mu sync.Mutex

	// 使用 func() error 以便支援依賴機制
	_, err = cron.Add("@every 30s", func() error {
		time.Sleep(2 * time.Second) // 任務執行時間超過超時限制
		return nil
	}, "超時測試", 500*time.Millisecond, func() {
		mu.Lock()
		timeoutTriggered = true
		mu.Unlock()
	})

	if err != nil {
		t.Fatalf("添加任務失敗: %v", err)
	}

	cron.Start()
	defer func() {
		ctx := cron.Stop()
		<-ctx.Done()
	}()

	// 等待 35 秒讓任務執行並觸發超時
	time.Sleep(35 * time.Second)

	mu.Lock()
	defer mu.Unlock()

	if !timeoutTriggered {
		t.Error("應該觸發超時回調")
	}
}

func TestTaskWithoutTimeout(t *testing.T) {
	config := goCron.Config{}

	cron, err := goCron.New(config)
	if err != nil {
		t.Fatalf("建立 cron 失敗: %v", err)
	}

	var taskCompleted bool
	var mu sync.Mutex

	// 使用 func() error 以便支援依賴機制
	_, err = cron.Add("@every 30s", func() error {
		time.Sleep(200 * time.Millisecond)
		mu.Lock()
		taskCompleted = true
		mu.Unlock()
		return nil
	}, "無延遲測試")

	if err != nil {
		t.Fatalf("添加任務失敗: %v", err)
	}

	cron.Start()
	defer func() {
		ctx := cron.Stop()
		<-ctx.Done()
	}()

	// 等待 35 秒讓任務執行
	time.Sleep(35 * time.Second)

	mu.Lock()
	if !taskCompleted {
		t.Error("任務應該正常完成")
	}
	mu.Unlock()
}

func TestTaskNormalCompletionWithDelay(t *testing.T) {
	config := goCron.Config{}

	cron, err := goCron.New(config)
	if err != nil {
		t.Fatalf("建立 cron 失敗: %v", err)
	}

	var taskCompleted bool
	var timeoutTriggered bool
	var mu sync.Mutex

	// 使用 func() error 以便支援依賴機制
	_, err = cron.Add("@every 30s", func() error {
		time.Sleep(100 * time.Millisecond)
		mu.Lock()
		taskCompleted = true
		mu.Unlock()
		return nil
	}, "正常完成測試", 500*time.Millisecond, func() {
		mu.Lock()
		timeoutTriggered = true
		mu.Unlock()
	})

	if err != nil {
		t.Fatalf("添加任務失敗: %v", err)
	}

	cron.Start()
	defer func() {
		ctx := cron.Stop()
		<-ctx.Done()
	}()

	// 等待 35 秒讓任務執行
	time.Sleep(35 * time.Second)

	mu.Lock()
	defer mu.Unlock()

	if !taskCompleted {
		t.Error("任務應該正常完成")
	}
	if timeoutTriggered {
		t.Error("不應該觸發超時回調")
	}
}

// 測試標準 cron 語法
func TestStandardCronSyntax(t *testing.T) {
	config := goCron.Config{}
	c, err := goCron.New(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// 測試標準 cron 語法
	_, err = c.Add("0 * * * *", func() {}, "Every hour")
	if err != nil {
		t.Errorf("Expected no error for valid cron syntax, got %v", err)
	}

	_, err = c.Add("30 2 * * *", func() {}, "Daily at 2:30 AM")
	if err != nil {
		t.Errorf("Expected no error for valid cron syntax, got %v", err)
	}

	_, err = c.Add("0 9 * * 1", func() {}, "Weekdays at 9 AM")
	if err != nil {
		t.Errorf("Expected no error for valid cron syntax, got %v", err)
	}
}

// 測試自定義描述符
func TestCustomDescriptors(t *testing.T) {
	config := goCron.Config{}
	c, err := goCron.New(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	descriptors := []string{
		"@yearly",
		"@annually",
		"@monthly",
		"@weekly",
		"@daily",
		"@midnight",
		"@hourly",
	}

	for _, desc := range descriptors {
		_, err = c.Add(desc, func() {}, fmt.Sprintf("Task for %s", desc))
		if err != nil {
			t.Errorf("Expected no error for descriptor %s, got %v", desc, err)
		}
	}
}

// 測試依賴功能
func TestCronWithDependencies(t *testing.T) {
	c, err := goCron.New(goCron.Config{})
	if err != nil {
		t.Fatalf("建立 cron 失敗: %v", err)
	}

	var execOrder []int64
	var mu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(3)

	addToOrder := func(id int64) {
		mu.Lock()
		execOrder = append(execOrder, id)
		mu.Unlock()
		wg.Done()
	}

	// 添加無依賴任務 - 使用 func() error
	task1ID, err := c.Add("@every 30s", func() error {
		addToOrder(1)
		return nil
	}, "task1")
	if err != nil {
		t.Fatalf("添加 task1 失敗: %v", err)
	}

	// 添加依賴 task1 的任務 - 使用 func() error
	task2ID, err := c.Add("@every 30s", func() error {
		addToOrder(2)
		return nil
	}, "task2", []int64{task1ID})
	if err != nil {
		t.Fatalf("添加 task2 失敗: %v", err)
	}

	// 添加依賴 task2 的任務 - 使用 func() error
	_, err = c.Add("@every 30s", func() error {
		addToOrder(3)
		return nil
	}, "task3", []int64{task2ID})
	if err != nil {
		t.Fatalf("添加 task3 失敗: %v", err)
	}

	c.Start()
	defer func() {
		ctx := c.Stop()
		<-ctx.Done()
	}()

	// 等待任務執行
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// 檢查執行順序
		if len(execOrder) != 3 {
			t.Fatalf("執行任務數量錯誤，預期 3，實際 %d", len(execOrder))
		}

		expectedOrder := []int64{1, 2, 3}
		for i, expected := range expectedOrder {
			if execOrder[i] != expected {
				t.Errorf("執行順序錯誤，位置 %d 預期 %d，實際 %d", i, expected, execOrder[i])
			}
		}

	case <-time.After(2 * time.Minute):
		t.Fatal("測試超時")
	}
}

// 測試 func() 不能有依賴
func TestFuncCannotHaveDependencies(t *testing.T) {
	c, err := goCron.New(goCron.Config{})
	if err != nil {
		t.Fatalf("建立 cron 失敗: %v", err)
	}

	// 先添加一個任務作為依賴
	depTaskID, err := c.Add("@every 30s", func() error {
		return nil
	}, "dependency_task")
	if err != nil {
		t.Fatalf("添加依賴任務失敗: %v", err)
	}

	// 嘗試讓 func() 有依賴，應該失敗
	_, err = c.Add("@every 30s", func() {
		// 無返回值函數
	}, "invalid_task", []int64{depTaskID})

	if err == nil {
		t.Error("func() 不應該能有依賴關係")
	}

	expectedError := "Need return value to get dependence support"
	if err.Error() != expectedError {
		t.Errorf("錯誤訊息不符，預期: %s，實際: %s", expectedError, err.Error())
	}
}

func TestCronTaskFailure(t *testing.T) {
	c, err := goCron.New(goCron.Config{})
	if err != nil {
		t.Fatalf("建立 cron 失敗: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(1)

	// 添加會失敗的任務 - 使用 func() error
	failTaskID, err := c.Add("@every 30s", func() error {
		wg.Done()
		return errors.New("模擬任務失敗")
	}, "fail_task")
	if err != nil {
		t.Fatalf("添加失敗任務失敗: %v", err)
	}

	// 添加依賴失敗任務的任務 - 使用 func() error
	var depTaskExecuted bool
	_, err = c.Add("@every 30s", func() error {
		depTaskExecuted = true
		return nil
	}, "dep_task", []int64{failTaskID})
	if err != nil {
		t.Fatalf("添加依賴任務失敗: %v", err)
	}

	c.Start()
	defer func() {
		ctx := c.Stop()
		<-ctx.Done()
	}()

	// 等待失敗任務執行
	done := make(chan struct{})
	go func() {
		wg.Wait()
		// 等待一段時間確保依賴任務不會執行
		time.Sleep(2 * time.Second)
		close(done)
	}()

	select {
	case <-done:
		if depTaskExecuted {
			t.Error("依賴任務不應該在前置任務失敗後執行")
		}
	case <-time.After(2 * time.Minute):
		t.Fatal("測試超時")
	}
}

func TestCronComplexDependencies(t *testing.T) {
	c, err := goCron.New(goCron.Config{})
	if err != nil {
		t.Fatalf("建立 cron 失敗: %v", err)
	}

	var execOrder []int64
	var mu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(5)

	addToOrder := func(id int64) {
		mu.Lock()
		execOrder = append(execOrder, id)
		mu.Unlock()
		wg.Done()
	}

	// 複雜依賴關係 - 所有任務都使用 func() error
	task1ID, _ := c.Add("@every 30s", func() error { addToOrder(1); return nil }, "task1")
	task2ID, _ := c.Add("@every 30s", func() error { addToOrder(2); return nil }, "task2")
	task3ID, _ := c.Add("@every 30s", func() error { addToOrder(3); return nil }, "task3", []int64{task1ID})
	task4ID, _ := c.Add("@every 30s", func() error { addToOrder(4); return nil }, "task4", []int64{task1ID, task2ID})
	_, _ = c.Add("@every 30s", func() error { addToOrder(5); return nil }, "task5", []int64{task3ID, task4ID})

	c.Start()
	defer func() {
		ctx := c.Stop()
		<-ctx.Done()
	}()

	// 等待所有任務執行
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// 檢查執行順序是否符合依賴關係
		orderMap := make(map[int64]int)
		for i, id := range execOrder {
			orderMap[id] = i
		}

		// task1 應該在 task3 和 task4 之前
		if orderMap[1] >= orderMap[3] || orderMap[1] >= orderMap[4] {
			t.Error("task1 應該在 task3 和 task4 之前執行")
		}

		// task2 應該在 task4 之前
		if orderMap[2] >= orderMap[4] {
			t.Error("task2 應該在 task4 之前執行")
		}

		// task3 和 task4 應該在 task5 之前
		if orderMap[3] >= orderMap[5] || orderMap[4] >= orderMap[5] {
			t.Error("task3 和 task4 應該在 task5 之前執行")
		}

	case <-time.After(3 * time.Minute):
		t.Fatal("測試超時")
	}
}
