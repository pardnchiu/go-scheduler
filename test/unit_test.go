/*
 * unit_test.go
 * this file is created by github copilot
 * to test the cron package
 */

package main

import (
	"fmt"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	golangCron "github.com/pardnchiu/go-cron"
)

func TestCronCreation(t *testing.T) {
	stdLogger := log.New(os.Stdout, "TEST: ", log.LstdFlags)
	config := golangCron.Config{
		Logger: golangCron.NewLoggerFromStdLogger(stdLogger),
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

func TestCronWithSilentLogger(t *testing.T) {
	config := golangCron.Config{
		Logger: &golangCron.NoOpLogger{},
	}
	c, err := golangCron.New(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if c == nil {
		t.Fatal("Expected cron instance, got nil")
	}
}

func TestCronWithWriterLogger(t *testing.T) {
	config := golangCron.Config{
		Logger: golangCron.NewLoggerFromWriter(os.Stderr),
	}
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
		Logger: &golangCron.NoOpLogger{},
	}
	c, err := golangCron.New(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	var mu sync.Mutex
	count := 0

	_, err = c.Add("@every 1s", func() {
		mu.Lock()
		count++
		mu.Unlock()
	})
	if err != nil {
		t.Fatalf("Expected no error adding task, got %v", err)
	}

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
		Logger: &golangCron.NoOpLogger{},
	}
	c, err := golangCron.New(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	executed := false
	_, err = c.Add("@every 1s", func() {
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
	config := golangCron.Config{
		Logger: &golangCron.NoOpLogger{},
	}
	c, err := golangCron.New(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	var mu sync.Mutex
	count1 := 0
	count2 := 0

	_, err = c.Add("@every 1s", func() {
		mu.Lock()
		count1++
		mu.Unlock()
	}, "Task 1")
	if err != nil {
		t.Fatalf("Expected no error adding task 1, got %v", err)
	}

	_, err = c.Add("@every 2s", func() {
		mu.Lock()
		count2++
		mu.Unlock()
	})
	if err != nil {
		t.Fatalf("Expected no error adding task 2, got %v", err)
	}

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
		Logger: &golangCron.NoOpLogger{},
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
		Logger: &golangCron.NoOpLogger{},
	}
	c, err := golangCron.New(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	var mu sync.Mutex
	normalTaskExecuted := false

	_, err = c.Add("@every 1s", func() {
		panic("test panic")
	})
	if err != nil {
		t.Fatalf("Expected no error adding panic task, got %v", err)
	}

	_, err = c.Add("@every 1s", func() {
		mu.Lock()
		normalTaskExecuted = true
		mu.Unlock()
	})
	if err != nil {
		t.Fatalf("Expected no error adding normal task, got %v", err)
	}

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
	c, _ := golangCron.New(golangCron.Config{
		Logger: &golangCron.NoOpLogger{},
	})

	// 期望返回錯誤而不是 panic
	_, err := c.Add("invalid-cron-expression", func() {}, "Invalid Schedule Test")
	if err == nil {
		t.Error("Expected error for invalid schedule, but got none")
	}
}

func TestCronList(t *testing.T) {
	config := golangCron.Config{
		Logger: &golangCron.NoOpLogger{},
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
		Logger: &golangCron.NoOpLogger{},
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
		Logger: &golangCron.NoOpLogger{},
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
		Logger: &golangCron.NoOpLogger{},
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
		Logger: &golangCron.NoOpLogger{},
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
		Logger: &golangCron.NoOpLogger{},
	}
	c, err := golangCron.New(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// 新增任務
	_, err = c.Add("@every 1s", func() {}, "Running Task")
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

func TestDifferentLoggerTypes(t *testing.T) {
	// log.Logger
	stdLogger := log.New(os.Stdout, "CRON-TEST: ", log.LstdFlags)
	config1 := golangCron.Config{
		Logger: golangCron.NewLoggerFromStdLogger(stdLogger),
	}
	c1, err := golangCron.New(config1)
	if err != nil {
		t.Fatalf("Expected no error with standard logger, got %v", err)
	}
	if c1 == nil {
		t.Fatal("Expected cron instance with standard logger, got nil")
	}

	// io.Writer
	config2 := golangCron.Config{
		Logger: golangCron.NewLoggerFromWriter(os.Stderr),
	}
	c2, err := golangCron.New(config2)
	if err != nil {
		t.Fatalf("Expected no error with writer logger, got %v", err)
	}
	if c2 == nil {
		t.Fatal("Expected cron instance with writer logger, got nil")
	}

	// NoOp
	config3 := golangCron.Config{
		Logger: &golangCron.NoOpLogger{},
	}
	c3, err := golangCron.New(config3)
	if err != nil {
		t.Fatalf("Expected no error with noop logger, got %v", err)
	}
	if c3 == nil {
		t.Fatal("Expected cron instance with noop logger, got nil")
	}

	// (nil)
	config4 := golangCron.Config{
		Logger: nil,
	}
	c4, err := golangCron.New(config4)
	if err != nil {
		t.Fatalf("Expected no error with default logger, got %v", err)
	}
	if c4 == nil {
		t.Fatal("Expected cron instance with default logger, got nil")
	}
}
