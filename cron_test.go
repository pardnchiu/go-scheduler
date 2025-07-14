// cron_test.go - 放在主程式碼同目錄
package goCron

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test Configuration
var testConfig = Config{
	Location: time.Local,
}

// Helper Functions

func createTestCron(t *testing.T) *cron {
	t.Helper()

	c, err := New(testConfig)
	require.NoError(t, err, "Failed to create cron instance")
	require.NotNil(t, c, "Cron instance should not be nil")

	return c
}

func cleanupCron(t *testing.T, c *cron) {
	t.Helper()

	if c != nil {
		ctx := c.Stop()
		select {
		case <-ctx.Done():
		case <-time.After(5 * time.Second):
			t.Error("Cron cleanup timeout")
		}
	}
}

// Unit Tests

// TestCron_New 測試實例創建
func TestCron_New(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name:    "default config",
			config:  Config{},
			wantErr: false,
		},
		{
			name: "with location",
			config: Config{
				Location: time.UTC,
			},
			wantErr: false,
		},
		{
			name: "nil location should use default",
			config: Config{
				Location: nil,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := New(tt.config)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, c)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, c)

				// Verify default location is set
				if tt.config.Location == nil {
					assert.Equal(t, time.Local, c.location)
				} else {
					assert.Equal(t, tt.config.Location, c.location)
				}
			}
		})
	}
}

// TestCron_Initialization 測試基本初始化
func TestCron_Initialization(t *testing.T) {
	c := createTestCron(t)
	defer cleanupCron(t, c)

	assert.NotNil(t, c.heap, "Task heap should be initialized")
	assert.NotNil(t, c.parser, "Parser should be initialized")
	assert.NotNil(t, c.stop, "Stop channel should be initialized")
	assert.NotNil(t, c.add, "Add channel should be initialized")
	assert.NotNil(t, c.remove, "Remove channel should be initialized")
	assert.NotNil(t, c.removeAll, "RemoveAll channel should be initialized")
	assert.NotNil(t, c.depend, "Depend manager should be initialized")
	assert.False(t, c.running, "Should not be running initially")
}

// TestCron_Add 測試任務添加
func TestCron_Add(t *testing.T) {
	c := createTestCron(t)
	defer cleanupCron(t, c)

	tests := []struct {
		name     string
		spec     string
		action   interface{}
		args     []interface{}
		wantErr  bool
		errorMsg string
	}{
		{
			name:    "valid func() task",
			spec:    "@every 30s",
			action:  func() {},
			wantErr: false,
		},
		{
			name:    "valid func() error task",
			spec:    "@every 30s",
			action:  func() error { return nil },
			wantErr: false,
		},
		{
			name:    "valid task with description",
			spec:    "@every 30s",
			action:  func() {},
			args:    []interface{}{"test task"},
			wantErr: false,
		},
		{
			name:    "valid task with timeout",
			spec:    "@every 30s",
			action:  func() error { return nil },
			args:    []interface{}{5 * time.Second},
			wantErr: false,
		},
		{
			name:    "valid task with timeout callback",
			spec:    "@every 30s",
			action:  func() error { return nil },
			args:    []interface{}{5 * time.Second, func() {}},
			wantErr: false,
		},
		{
			name:     "invalid schedule",
			spec:     "invalid-cron-expression",
			action:   func() {},
			wantErr:  true,
			errorMsg: "Failed to parse",
		},
		{
			name:     "too frequent interval",
			spec:     "@every 1s",
			action:   func() {},
			wantErr:  true,
			errorMsg: "minimum interval is 30s",
		},
		{
			name:     "invalid action type",
			spec:     "@every 30s",
			action:   "not a function",
			wantErr:  true,
			errorMsg: "Action need to be func()",
		},
		{
			name:     "func() with dependencies should fail",
			spec:     "@every 30s",
			action:   func() {},
			args:     []interface{}{[]int64{1}},
			wantErr:  true,
			errorMsg: "Need return value to get dependence support",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := c.Add(tt.spec, tt.action, tt.args...)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Equal(t, int64(0), id)
			} else {
				assert.NoError(t, err)
				assert.Greater(t, id, int64(0))
			}
		})
	}
}

// TestCron_StandardCronSyntax 測試標準 cron 語法
func TestCron_StandardCronSyntax(t *testing.T) {
	c := createTestCron(t)
	defer cleanupCron(t, c)

	validExpressions := []string{
		"0 * * * *",     // Every hour
		"30 2 * * *",    // Daily at 2:30 AM
		"0 9 * * 1",     // Mondays at 9 AM
		"*/15 * * * *",  // Every 15 minutes
		"0 0 1 * *",     // First day of month
		"0 0 * * 1-5",   // Weekdays
		"0 0 * * 1,3,5", // Mon, Wed, Fri
	}

	for _, expr := range validExpressions {
		t.Run(fmt.Sprintf("cron_%s", expr), func(t *testing.T) {
			id, err := c.Add(expr, func() {}, fmt.Sprintf("Task for %s", expr))
			assert.NoError(t, err, "Should accept valid cron expression: %s", expr)
			assert.Greater(t, id, int64(0))
		})
	}
}

// TestCron_CustomDescriptors 測試自定義描述符
func TestCron_CustomDescriptors(t *testing.T) {
	c := createTestCron(t)
	defer cleanupCron(t, c)

	descriptors := []string{
		"@yearly",
		"@annually",
		"@monthly",
		"@weekly",
		"@daily",
		"@midnight",
		"@hourly",
		"@every 30s",
		"@every 5m",
		"@every 2h",
	}

	for _, desc := range descriptors {
		t.Run(fmt.Sprintf("descriptor_%s", desc), func(t *testing.T) {
			id, err := c.Add(desc, func() {}, fmt.Sprintf("Task for %s", desc))
			assert.NoError(t, err, "Should accept descriptor: %s", desc)
			assert.Greater(t, id, int64(0))
		})
	}
}

// TestCron_StartStop 測試啟動和停止
func TestCron_StartStop(t *testing.T) {
	c := createTestCron(t)

	// Test start
	c.Start()
	assert.True(t, c.running, "Should be running after start")

	// Test double start (should be safe)
	c.Start()
	assert.True(t, c.running, "Should still be running after double start")

	// Test stop
	ctx := c.Stop()
	assert.False(t, c.running, "Should not be running after stop")

	// Wait for stop to complete
	select {
	case <-ctx.Done():
		// Success
	case <-time.After(2 * time.Second):
		t.Error("Stop should complete within 2 seconds")
	}
}

// TestCron_StopWithoutStart 測試未啟動時停止
func TestCron_StopWithoutStart(t *testing.T) {
	c := createTestCron(t)

	ctx := c.Stop()

	select {
	case <-ctx.Done():
		// Should complete immediately
	case <-time.After(100 * time.Millisecond):
		t.Error("Stop should complete immediately when not started")
	}
}

// TestCron_List 測試任務列表
func TestCron_List(t *testing.T) {
	c := createTestCron(t)
	defer cleanupCron(t, c)

	// Initially empty
	tasks := c.List()
	assert.Len(t, tasks, 0, "Should start with empty task list")

	// Add tasks
	id1, err := c.Add("@every 30s", func() {}, "Task 1")
	require.NoError(t, err)

	id2, err := c.Add("@every 60s", func() {}, "Task 2")
	require.NoError(t, err)

	// Check list
	tasks = c.List()
	assert.Len(t, tasks, 2, "Should have 2 tasks")

	// Verify task IDs
	foundTask1 := false
	foundTask2 := false
	for _, task := range tasks {
		if task.ID == id1 {
			foundTask1 = true
			assert.Equal(t, "Task 1", task.description)
		}
		if task.ID == id2 {
			foundTask2 = true
			assert.Equal(t, "Task 2", task.description)
		}
	}
	assert.True(t, foundTask1, "Task 1 should be in list")
	assert.True(t, foundTask2, "Task 2 should be in list")
}

// TestCron_Remove 測試任務移除
func TestCron_Remove(t *testing.T) {
	c := createTestCron(t)
	defer cleanupCron(t, c)

	// Add tasks
	id1, err := c.Add("@every 30s", func() {}, "Task 1")
	require.NoError(t, err)

	id2, err := c.Add("@every 60s", func() {}, "Task 2")
	require.NoError(t, err)

	// Remove one task
	c.Remove(id1)

	// Check remaining tasks
	tasks := c.List()
	assert.Len(t, tasks, 1, "Should have 1 task after removal")
	assert.Equal(t, id2, tasks[0].ID, "Wrong task remained after removal")
}

// TestCron_RemoveAll 測試移除所有任務
func TestCron_RemoveAll(t *testing.T) {
	c := createTestCron(t)
	defer cleanupCron(t, c)

	// Add multiple tasks
	_, err := c.Add("@every 30s", func() {}, "Task 1")
	require.NoError(t, err)

	_, err = c.Add("@every 60s", func() {}, "Task 2")
	require.NoError(t, err)

	_, err = c.Add("@every 90s", func() {}, "Task 3")
	require.NoError(t, err)

	// Verify tasks added
	tasks := c.List()
	assert.Len(t, tasks, 3, "Should have 3 tasks")

	// Remove all tasks
	c.RemoveAll()

	// Verify all tasks removed
	tasks = c.List()
	assert.Len(t, tasks, 0, "Should have 0 tasks after RemoveAll")
}

// TestCron_TaskExecution 測試任務執行
func TestCron_TaskExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping execution test in short mode")
	}

	c := createTestCron(t)
	defer cleanupCron(t, c)

	var mu sync.Mutex
	var executed bool

	_, err := c.Add("@every 30s", func() {
		mu.Lock()
		executed = true
		mu.Unlock()
	}, "Execution test")
	require.NoError(t, err)

	c.Start()

	// Wait for execution
	timeout := time.After(35 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			t.Fatal("Task should have executed within 35 seconds")
		case <-ticker.C:
			mu.Lock()
			isExecuted := executed
			mu.Unlock()

			if isExecuted {
				return // Test passed
			}
		}
	}
}

// TestCron_MultipleTaskExecution 測試多任務執行
func TestCron_MultipleTaskExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping execution test in short mode")
	}

	c := createTestCron(t)
	defer cleanupCron(t, c)

	var mu sync.Mutex
	count1 := 0
	count2 := 0

	_, err := c.Add("@every 30s", func() {
		mu.Lock()
		count1++
		mu.Unlock()
	}, "Task 1")
	require.NoError(t, err)

	_, err = c.Add("@every 30s", func() {
		mu.Lock()
		count2++
		mu.Unlock()
	}, "Task 2")
	require.NoError(t, err)

	c.Start()

	// Wait for execution
	time.Sleep(35 * time.Second)

	mu.Lock()
	finalCount1 := count1
	finalCount2 := count2
	mu.Unlock()

	assert.GreaterOrEqual(t, finalCount1, 1, "Task 1 should execute at least once")
	assert.GreaterOrEqual(t, finalCount2, 1, "Task 2 should execute at least once")
}

// TestCron_TaskPanicRecovery 測試 panic 恢復
func TestCron_TaskPanicRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping execution test in short mode")
	}

	c := createTestCron(t)
	defer cleanupCron(t, c)

	var mu sync.Mutex
	normalTaskExecuted := false

	// Add panic task
	_, err := c.Add("@every 30s", func() {
		panic("test panic")
	}, "Panic task")
	require.NoError(t, err)

	// Add normal task
	_, err = c.Add("@every 30s", func() {
		mu.Lock()
		normalTaskExecuted = true
		mu.Unlock()
	}, "Normal task")
	require.NoError(t, err)

	c.Start()

	// Wait for execution
	time.Sleep(35 * time.Second)

	mu.Lock()
	executed := normalTaskExecuted
	mu.Unlock()

	assert.True(t, executed, "Normal task should execute even when other task panics")
}

// TestCron_TaskTimeout 測試任務超時
func TestCron_TaskTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping execution test in short mode")
	}

	c := createTestCron(t)
	defer cleanupCron(t, c)

	var mu sync.Mutex
	timeoutTriggered := false

	_, err := c.Add("@every 30s", func() error {
		time.Sleep(2 * time.Second) // Longer than timeout
		return nil
	}, "Timeout test", 500*time.Millisecond, func() {
		mu.Lock()
		timeoutTriggered = true
		mu.Unlock()
	})
	require.NoError(t, err)

	c.Start()

	// Wait for timeout
	time.Sleep(35 * time.Second)

	mu.Lock()
	triggered := timeoutTriggered
	mu.Unlock()

	assert.True(t, triggered, "Timeout callback should be triggered")
}

// TestCron_TaskWithoutTimeout 測試無超時任務
func TestCron_TaskWithoutTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping execution test in short mode")
	}

	c := createTestCron(t)
	defer cleanupCron(t, c)

	var mu sync.Mutex
	taskCompleted := false

	_, err := c.Add("@every 30s", func() error {
		time.Sleep(200 * time.Millisecond)
		mu.Lock()
		taskCompleted = true
		mu.Unlock()
		return nil
	}, "No timeout test")
	require.NoError(t, err)

	c.Start()

	// Wait for completion
	time.Sleep(35 * time.Second)

	mu.Lock()
	completed := taskCompleted
	mu.Unlock()

	assert.True(t, completed, "Task should complete normally")
}

// TestCron_Dependencies 測試任務依賴
func TestCron_Dependencies(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping dependency test in short mode")
	}

	c := createTestCron(t)
	defer cleanupCron(t, c)

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

	// Task 1 - no dependencies
	task1ID, err := c.Add("@every 30s", func() error {
		addToOrder(1)
		return nil
	}, "task1")
	require.NoError(t, err)

	// Task 2 - depends on task1
	task2ID, err := c.Add("@every 30s", func() error {
		addToOrder(2)
		return nil
	}, "task2", []int64{task1ID})
	require.NoError(t, err)

	// Task 3 - depends on task2
	_, err = c.Add("@every 30s", func() error {
		addToOrder(3)
		return nil
	}, "task3", []int64{task2ID})
	require.NoError(t, err)

	c.Start()

	// Wait for execution
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Check execution order
		mu.Lock()
		order := make([]int64, len(execOrder))
		copy(order, execOrder)
		mu.Unlock()

		assert.Len(t, order, 3, "All tasks should execute")

		expectedOrder := []int64{1, 2, 3}
		assert.Equal(t, expectedOrder, order, "Tasks should execute in dependency order")

	case <-time.After(2 * time.Minute):
		t.Fatal("Dependency test timeout")
	}
}

// TestCron_TaskFailureDependency 測試任務失敗對依賴的影響
func TestCron_TaskFailureDependency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping dependency test in short mode")
	}

	c := createTestCron(t)
	defer cleanupCron(t, c)

	var wg sync.WaitGroup
	wg.Add(1)

	var mu sync.Mutex
	depTaskExecuted := false

	// Failing task
	failTaskID, err := c.Add("@every 30s", func() error {
		wg.Done()
		return errors.New("simulated failure")
	}, "fail_task")
	require.NoError(t, err)

	// Dependent task
	_, err = c.Add("@every 30s", func() error {
		mu.Lock()
		depTaskExecuted = true
		mu.Unlock()
		return nil
	}, "dep_task", []int64{failTaskID})
	require.NoError(t, err)

	c.Start()

	// Wait for failure
	done := make(chan struct{})
	go func() {
		wg.Wait()
		time.Sleep(2 * time.Second) // Wait to ensure dependent task doesn't execute
		close(done)
	}()

	select {
	case <-done:
		mu.Lock()
		executed := depTaskExecuted
		mu.Unlock()

		assert.False(t, executed, "Dependent task should not execute after prerequisite fails")

	case <-time.After(2 * time.Minute):
		t.Fatal("Failure test timeout")
	}
}

// TestCron_ComplexDependencies 測試複雜依賴關係
func TestCron_ComplexDependencies(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping complex dependency test in short mode")
	}

	c := createTestCron(t)
	defer cleanupCron(t, c)

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

	// Complex dependency graph
	task1ID, _ := c.Add("@every 30s", func() error { addToOrder(1); return nil }, "task1")
	task2ID, _ := c.Add("@every 30s", func() error { addToOrder(2); return nil }, "task2")
	task3ID, _ := c.Add("@every 30s", func() error { addToOrder(3); return nil }, "task3", []int64{task1ID})
	task4ID, _ := c.Add("@every 30s", func() error { addToOrder(4); return nil }, "task4", []int64{task1ID, task2ID})
	_, _ = c.Add("@every 30s", func() error { addToOrder(5); return nil }, "task5", []int64{task3ID, task4ID})

	c.Start()

	// Wait for execution
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Check execution order constraints
		mu.Lock()
		orderMap := make(map[int64]int)
		for i, id := range execOrder {
			orderMap[id] = i
		}
		mu.Unlock()

		// task1 should execute before task3 and task4
		assert.Less(t, orderMap[1], orderMap[3], "task1 should execute before task3")
		assert.Less(t, orderMap[1], orderMap[4], "task1 should execute before task4")

		// task2 should execute before task4
		assert.Less(t, orderMap[2], orderMap[4], "task2 should execute before task4")

		// task3 and task4 should execute before task5
		assert.Less(t, orderMap[3], orderMap[5], "task3 should execute before task5")
		assert.Less(t, orderMap[4], orderMap[5], "task4 should execute before task5")

	case <-time.After(3 * time.Minute):
		t.Fatal("Complex dependency test timeout")
	}
}

// Benchmark Tests

// BenchmarkCron_Add 添加任務的效能測試
func BenchmarkCron_Add(b *testing.B) {
	c, _ := New(testConfig)
	defer cleanupCron(&testing.T{}, c)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := c.Add("@every 30s", func() {}, fmt.Sprintf("task-%d", i))
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCron_List 列出任務的效能測試
func BenchmarkCron_List(b *testing.B) {
	c, _ := New(testConfig)
	defer cleanupCron(&testing.T{}, c)

	// Add some tasks
	for i := 0; i < 100; i++ {
		c.Add("@every 30s", func() {}, fmt.Sprintf("task-%d", i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.List()
	}
}
