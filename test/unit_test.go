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
	})

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
	})

	c.Add("@every 2s", func() {
		mu.Lock()
		count2++
		mu.Unlock()
	})

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
	_, err := c.Add("invalid-cron-expression", func() {})
	if err == nil {
		t.Error("Expected error for invalid schedule, but got none")
	}
}
