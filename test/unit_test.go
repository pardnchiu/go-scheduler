/*
 * unit_test.go
 * this file is created by github copilot
 * to test the cron package
 */

package main

import (
	"fmt"
	"sync"
	"testing"
	"time"

	goCron "github.com/pardnchiu/go-cron"
	golangCron "github.com/pardnchiu/go-cron"
)

func TestCronCreation(t *testing.T) {
	config := golangCron.Config{
		Log: &golangCron.Log{
			Stdout: true,
		},
	}
	c, err := golangCron.New(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if c == nil {
		t.Fatal("Expected cron instance, got nil")
	}
}

func TestCronWithoutLog(t *testing.T) {
	config := golangCron.Config{}
	c, err := golangCron.New(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if c == nil {
		t.Fatal("Expected cron instance, got nil")
	}
}

func TestCronEverySecond(t *testing.T) {
	config := golangCron.Config{
		Log: &golangCron.Log{
			Stdout: false,
		},
	}
	c, err := golangCron.New(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	var mu sync.Mutex
	count := 0

	c.Add("@every 1s", func() {
		mu.Lock()
		count++
		mu.Unlock()
	})

	c.Start()

	time.Sleep(3*time.Second + 100*time.Millisecond)

	ctx := c.Stop()
	<-ctx.Done()

	mu.Lock()
	finalCount := count
	mu.Unlock()

	if finalCount < 2 || finalCount > 4 {
		t.Fatalf("Expected count to be 2-4, got %d", finalCount)
	}
}

func TestCronStartStop(t *testing.T) {
	config := golangCron.Config{
		Log: &golangCron.Log{
			Stdout: false,
		},
	}
	c, err := golangCron.New(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	executed := false
	c.Add("@every 1s", func() {
		executed = true
		fmt.Print("Task executed\n", time.Now().Format("15:04:05"), "\n", executed)
	}, "Test Task")

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
	config := golangCron.Config{
		Log: &golangCron.Log{
			Stdout: false,
		},
	}
	c, err := golangCron.New(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	var mu sync.Mutex
	count1 := 0
	count2 := 0

	c.Add("@every 1s", func() {
		mu.Lock()
		count1++
		mu.Unlock()
	}, "Task 1")

	c.Add("@every 2s", func() {
		mu.Lock()
		count2++
		mu.Unlock()
	})

	c.Start()

	time.Sleep(3*time.Second + 100*time.Millisecond)

	ctx := c.Stop()
	<-ctx.Done()

	mu.Lock()
	finalCount1 := count1
	finalCount2 := count2
	mu.Unlock()

	if finalCount1 < 2 {
		t.Fatalf("Expected count1 to be at least 2, got %d", finalCount1)
	}
	if finalCount2 < 1 {
		t.Fatalf("Expected count2 to be at least 1, got %d", finalCount2)
	}
}

func TestCronStopWithoutStart(t *testing.T) {
	config := golangCron.Config{
		Log: &golangCron.Log{
			Stdout: false,
		},
	}
	c, err := golangCron.New(config)
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
	config := golangCron.Config{
		Log: &golangCron.Log{
			Stdout: false,
		},
	}
	c, err := golangCron.New(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	var mu sync.Mutex
	normalTaskExecuted := false

	c.Add("@every 1s", func() {
		panic("test panic")
	})

	c.Add("@every 1s", func() {
		mu.Lock()
		normalTaskExecuted = true
		mu.Unlock()
	})

	c.Start()

	time.Sleep(2*time.Second + 100*time.Millisecond)

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
	c, _ := golangCron.New(golangCron.Config{})

	// 期望返回錯誤而不是 panic
	_, err := c.Add("invalid-cron-expression", func() {}, "Invalid Schedule Test")
	if err == nil {
		t.Error("Expected error for invalid schedule, but got none")
	}
}

func TestCronList(t *testing.T) {
	config := golangCron.Config{
		Log: &golangCron.Log{
			Stdout: false,
		},
	}
	c, err := golangCron.New(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// 新增多個任務
	id1, err := c.Add("@every 1s", func() {}, "Task 1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	id2, err := c.Add("@every 2s", func() {}, "Task 2")
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
		if task.ID == id1 && task.Description == "Task 1" {
			foundTask1 = true
		}
		if task.ID == id2 && task.Description == "Task 2" {
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
	config := golangCron.Config{
		Log: &golangCron.Log{
			Stdout: false,
		},
	}
	c, err := golangCron.New(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// 新增任務
	id1, _ := c.Add("@every 1s", func() {}, "Task 1")
	id2, _ := c.Add("@every 2s", func() {}, "Task 2")

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
	config := golangCron.Config{
		Log: &golangCron.Log{
			Stdout: false,
		},
	}
	c, err := golangCron.New(config)
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
	config := golangCron.Config{
		Log: &golangCron.Log{
			Stdout: false,
		},
	}
	c, err := golangCron.New(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// 新增多個任務
	_, err = c.Add("@every 1s", func() {}, "Task 1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	_, err = c.Add("@every 2s", func() {}, "Task 2")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	_, err = c.Add("@every 3s", func() {}, "Task 3")
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
	config := golangCron.Config{
		Log: &golangCron.Log{
			Stdout: false,
		},
	}
	c, err := golangCron.New(config)
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
	config := golangCron.Config{
		Log: &golangCron.Log{
			Stdout: false,
		},
	}
	c, err := golangCron.New(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// 新增任務
	c.Add("@every 1s", func() {}, "Running Task")

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
	config := goCron.Config{
		Log: &goCron.Log{
			Path:   "./test.log",
			Stdout: false,
		},
	}

	cron, err := goCron.New(config)
	if err != nil {
		t.Fatalf("建立 cron 失敗: %v", err)
	}

	var timeoutTriggered bool
	var mu sync.Mutex

	// 使用 @every 1s 確保任務會立即開始
	_, err = cron.Add("@every 1s", func() {
		time.Sleep(2 * time.Second) // 任務執行時間超過超時限制
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

	// 等待足夠時間讓任務執行並觸發超時
	time.Sleep(3 * time.Second)

	mu.Lock()
	defer mu.Unlock()

	if !timeoutTriggered {
		t.Error("應該觸發超時回調")
	}
}

func TestTaskWithoutTimeout(t *testing.T) {
	config := goCron.Config{
		Log: &goCron.Log{
			Path:   "./test.log",
			Stdout: false,
		},
	}

	cron, err := goCron.New(config)
	if err != nil {
		t.Fatalf("建立 cron 失敗: %v", err)
	}

	var taskCompleted bool
	var mu sync.Mutex

	// 使用 @every 1s 確保任務會立即開始
	_, err = cron.Add("@every 1s", func() {
		time.Sleep(200 * time.Millisecond)
		mu.Lock()
		taskCompleted = true
		mu.Unlock()
	}, "無延遲測試")

	if err != nil {
		t.Fatalf("添加任務失敗: %v", err)
	}

	cron.Start()
	defer func() {
		ctx := cron.Stop()
		<-ctx.Done()
	}()

	time.Sleep(2 * time.Second)

	mu.Lock()
	if !taskCompleted {
		t.Error("任務應該正常完成")
	}
	mu.Unlock()
}

func TestTaskNormalCompletionWithDelay(t *testing.T) {
	config := goCron.Config{
		Log: &goCron.Log{
			Path:   "./test.log",
			Stdout: false,
		},
	}

	cron, err := goCron.New(config)
	if err != nil {
		t.Fatalf("建立 cron 失敗: %v", err)
	}

	var taskCompleted bool
	var timeoutTriggered bool
	var mu sync.Mutex

	// 使用 @every 1s 確保任務會立即開始
	_, err = cron.Add("@every 1s", func() {
		time.Sleep(100 * time.Millisecond)
		mu.Lock()
		taskCompleted = true
		mu.Unlock()
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

	time.Sleep(2 * time.Second)

	mu.Lock()
	defer mu.Unlock()

	if !taskCompleted {
		t.Error("任務應該正常完成")
	}
	if timeoutTriggered {
		t.Error("不應該觸發超時回調")
	}
}
