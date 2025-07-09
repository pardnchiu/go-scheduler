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
	config := golangCron.Config{}
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

func TestCronEveryThirtySeconds(t *testing.T) {
	config := golangCron.Config{}
	c, err := golangCron.New(config)
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
	config := golangCron.Config{}
	c, err := golangCron.New(config)
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
	config := golangCron.Config{}
	c, err := golangCron.New(config)
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
	config := golangCron.Config{}
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
	config := golangCron.Config{}
	c, err := golangCron.New(config)
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
	c, _ := golangCron.New(golangCron.Config{})

	// 期望返回錯誤而不是 panic
	_, err := c.Add("invalid-cron-expression", func() {}, "Invalid Schedule Test")
	if err == nil {
		t.Error("Expected error for invalid schedule, but got none")
	}
}

func TestCronInvalidMinInterval(t *testing.T) {
	c, _ := golangCron.New(golangCron.Config{})

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
	config := golangCron.Config{}
	c, err := golangCron.New(config)
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
	config := golangCron.Config{}
	c, err := golangCron.New(config)
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
	config := golangCron.Config{}
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
	config := golangCron.Config{}
	c, err := golangCron.New(config)
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
	config := golangCron.Config{}
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
	config := golangCron.Config{}
	c, err := golangCron.New(config)
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

	// 使用 @every 30s 確保任務會執行
	_, err = cron.Add("@every 30s", func() {
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

	// 使用 @every 30s 確保任務會執行
	_, err = cron.Add("@every 30s", func() {
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

	// 使用 @every 30s 確保任務會執行
	_, err = cron.Add("@every 30s", func() {
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
	config := golangCron.Config{}
	c, err := golangCron.New(config)
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
	config := golangCron.Config{}
	c, err := golangCron.New(config)
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

// 測試範圍語法
func TestRangeSyntax(t *testing.T) {
	config := goCron.Config{}
	c, err := goCron.New(config)
	if err != nil {
		t.Fatalf("建立 cron 失敗: %v", err)
	}

	testCases := []struct {
		spec        string
		description string
		shouldError bool
	}{
		{"0 9-17 * * *", "工作時間 9-17 點", false},
		{"0 0 1-5 * *", "每月 1-5 日", false},
		{"0 0 * * 1-5", "星期一到五", false},
		{"0-30 * * * *", "每小時 0-30 分", false},
		{"0 9-5 * * *", "無效範圍（開始大於結束）", true},
		{"0 25-30 * * *", "超出範圍", true},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			_, err := c.Add(tc.spec, func() {}, tc.description)
			if tc.shouldError && err == nil {
				t.Errorf("預期錯誤但未發生: %s", tc.spec)
			}
			if !tc.shouldError && err != nil {
				t.Errorf("預期成功但發生錯誤: %s, 錯誤: %v", tc.spec, err)
			}
		})
	}
}

// 測試列表語法
func TestListSyntax(t *testing.T) {
	config := goCron.Config{}
	c, err := goCron.New(config)
	if err != nil {
		t.Fatalf("建立 cron 失敗: %v", err)
	}

	testCases := []struct {
		spec        string
		description string
		shouldError bool
	}{
		{"0 0 * * 1,3,5", "星期一三五", false},
		{"0,15,30,45 * * * *", "每 15 分鐘", false},
		{"0 9,12,18 * * *", "9 點、12 點、18 點", false},
		{"0 0 1,15 * *", "每月 1 日和 15 日", false},
		{"0 0 * 1,3,5,7,9,11 *", "奇數月份", false},
		{"0 25,30 * * *", "超出範圍", true},
		{"0 0 * * 8,9", "星期超出範圍", true},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			_, err := c.Add(tc.spec, func() {}, tc.description)
			if tc.shouldError && err == nil {
				t.Errorf("預期錯誤但未發生: %s", tc.spec)
			}
			if !tc.shouldError && err != nil {
				t.Errorf("預期成功但發生錯誤: %s, 錯誤: %v", tc.spec, err)
			}
		})
	}
}

// 測試複合語法
func TestComplexSyntax(t *testing.T) {
	config := goCron.Config{}
	c, err := goCron.New(config)
	if err != nil {
		t.Fatalf("建立 cron 失敗: %v", err)
	}

	testCases := []struct {
		spec        string
		description string
		shouldError bool
	}{
		{"0 0 * * 1-3,5", "星期一到三，加星期五", false},
		{"0,30 9-17 * * 1-5", "工作日工作時間每半小時", false},
		{"15,45 8-10,14-16 * * *", "上午和下午特定時間", false},
		{"0 0 1-5,15,25-31 * *", "月初、月中、月末", false},
		{"0-15,30-45 * * * *", "每小時前 15 分和後 15 分", false},
		{"0 0 * * 1-3,8", "無效星期組合", true},
		{"0,70 * * * *", "分鐘超出範圍", true},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			_, err := c.Add(tc.spec, func() {}, tc.description)
			if tc.shouldError && err == nil {
				t.Errorf("預期錯誤但未發生: %s", tc.spec)
			}
			if !tc.shouldError && err != nil {
				t.Errorf("預期成功但發生錯誤: %s, 錯誤: %v", tc.spec, err)
			}
		})
	}
}

// 測試實際執行
func TestEnhancedSyntaxExecution(t *testing.T) {
	config := goCron.Config{}
	c, err := goCron.New(config)
	if err != nil {
		t.Fatalf("建立 cron 失敗: %v", err)
	}

	var mu sync.Mutex
	executedTasks := make(map[string]int)

	// 添加測試任務
	_, err = c.Add("@every 30s", func() {
		mu.Lock()
		executedTasks["every_30s"]++
		mu.Unlock()
	}, "每 30 秒執行")

	if err != nil {
		t.Fatalf("添加任務失敗: %v", err)
	}

	c.Start()
	defer func() {
		ctx := c.Stop()
		<-ctx.Done()
	}()

	// 等待 35 秒確保任務執行
	time.Sleep(35 * time.Second)

	mu.Lock()
	count := executedTasks["every_30s"]
	mu.Unlock()

	if count < 1 {
		t.Errorf("任務應至少執行 1 次，實際執行 %d 次", count)
	}
}

// 測試語法解析邊界情況
func TestSyntaxParsing(t *testing.T) {
	config := goCron.Config{}
	c, err := goCron.New(config)
	if err != nil {
		t.Fatalf("建立 cron 失敗: %v", err)
	}

	testCases := []struct {
		spec        string
		description string
		shouldError bool
	}{
		{"0 0 * * 0-6", "所有星期", false},
		{"0 0-23 * * *", "所有小時", false},
		{"0-59 * * * *", "所有分鐘", false},
		{"* * 1-31 * *", "所有日期", false},
		{"* * * 1-12 *", "所有月份", false},
		{"0 0 * * ", "缺少欄位", true},
		{"0 0 * * * *", "過多欄位", true},
		{"0 0 * * 1-", "不完整範圍", true},
		{"0 0 * * ,1,2", "以逗號開頭", true},
		{"0 0 * * 1,,2", "連續逗號", true},
		{"0 0 * * 1,2,", "以逗號結尾", true},
		{"0 0 * * -1,2", "以連字符開頭", true},
		{"0 0 * * 1--2", "連續連字符", true},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			_, err := c.Add(tc.spec, func() {}, tc.description)
			if tc.shouldError && err == nil {
				t.Errorf("預期錯誤但未發生: %s", tc.spec)
			}
			if !tc.shouldError && err != nil {
				t.Errorf("預期成功但發生錯誤: %s, 錯誤: %v", tc.spec, err)
			}
		})
	}
}
