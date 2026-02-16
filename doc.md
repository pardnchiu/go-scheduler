# go-scheduler - Documentation

> Back to [README](./README.md)

## Prerequisites

- Go 1.23 or higher

## Installation

### Using go get

```bash
go get github.com/pardnchiu/go-scheduler
```

### From Source

```bash
git clone https://github.com/pardnchiu/go-scheduler.git
cd go-scheduler
go build ./...
```

## Usage

### Basic

Create a scheduler and add periodic tasks:

```go
package main

import (
	"fmt"
	"log"

	goCron "github.com/pardnchiu/go-scheduler"
)

func main() {
	c, err := goCron.New(goCron.Config{})
	if err != nil {
		log.Fatal(err)
	}

	// Run every 30 seconds
	_, err = c.Add("@every 30s", func() {
		fmt.Println("runs every 30 seconds")
	}, "periodic task")
	if err != nil {
		log.Fatal(err)
	}

	c.Start()
	defer c.Stop()

	// Keep the program running
	select {}
}
```

### Standard Cron Syntax

```go
// Run daily at 2:30 AM
c.Add("30 2 * * *", func() {
	fmt.Println("daily backup")
}, "daily backup task")

// Run every Monday at 9 AM
c.Add("0 9 * * 1", func() {
	fmt.Println("weekly report")
}, "weekly report task")

// Run every 15 minutes
c.Add("*/15 * * * *", func() {
	fmt.Println("health check")
}, "health check")
```

### Tasks with Timeout Control

```go
import "time"

id, err := c.Add("@every 1m", func() error {
	// Long-running task
	time.Sleep(10 * time.Second)
	return nil
}, "timeout task", 5*time.Second, func() {
	// Callback triggered on timeout
	fmt.Println("task timed out, running cleanup")
})
```

### Task Dependency Chains

```go
// Task 1: no dependencies
task1ID, _ := c.Add("@every 1m", func() error {
	fmt.Println("task 1 executed")
	return nil
}, "task 1")

// Task 2: depends on task 1, skip on failure
task2ID, _ := c.Add("@every 1m", func() error {
	fmt.Println("task 2 executed")
	return nil
}, "task 2", []goCron.Wait{
	{ID: task1ID, State: goCron.Skip},
})

// Task 3: depends on task 2, stop on failure, custom 10s timeout
_, _ = c.Add("@every 1m", func() error {
	fmt.Println("task 3 executed")
	return nil
}, "task 3", []goCron.Wait{
	{ID: task2ID, Delay: 10 * time.Second, State: goCron.Stop},
})
```

### Dynamic Task Management

```go
// Add a task
id, _ := c.Add("@every 30s", func() {
	fmt.Println("removable task")
}, "temporary task")

// Remove a single task
c.Remove(id)

// Remove all tasks
c.RemoveAll()

// List current tasks
tasks := c.List()
for _, t := range tasks {
	fmt.Printf("Task ID: %d\n", t.ID)
}
```

## API Reference

### New

```go
func New(c Config) (*cron, error)
```

Create a new scheduler instance. When `Config.Location` is nil, the system local timezone is used. Logs are written to syslog when available, falling back to stderr.

### Config

```go
type Config struct {
	Location *time.Location
}
```

Scheduler configuration. `Location` specifies the timezone for schedule calculations.

### Add

```go
func (c *cron) Add(spec string, action interface{}, arg ...interface{}) (int64, error)
```

Add a scheduled task. `spec` accepts standard 5-field cron expressions or descriptors. `action` accepts `func()` or `func() error`. Optional arguments in order:

| Argument Type | Description |
|---------------|-------------|
| `string` | Task description |
| `time.Duration` | Task execution timeout |
| `func()` | Callback triggered on timeout |
| `[]Wait` | Dependency task list |

Returns the task ID and error. Using `func()` as the action does not support dependencies; use `func() error` instead.

### Wait

```go
type Wait struct {
	ID    int64
	Delay time.Duration
	State WaitState
}
```

Define a task dependency. `ID` is the prerequisite task ID, `Delay` is the timeout for waiting on the prerequisite (defaults to 1 minute), and `State` is the strategy when the prerequisite fails.

### WaitState

```go
const (
	Stop WaitState = iota  // Abort when prerequisite fails
	Skip                   // Skip failure and continue
)
```

### Start

```go
func (c *cron) Start()
```

Start the scheduler. Calling multiple times is safe and does not create additional scheduling loops.

### Stop

```go
func (c *cron) Stop() context.Context
```

Stop the scheduler and return a Context that is cancelled when all running tasks complete.

### Remove

```go
func (c *cron) Remove(id int64)
```

Remove the task with the specified ID.

### RemoveAll

```go
func (c *cron) RemoveAll()
```

Remove all tasks.

### List

```go
func (c *cron) List() []task
```

Return all enabled tasks.

### Schedule Syntax

| Format | Example | Description |
|--------|---------|-------------|
| 5-field cron | `30 2 * * *` | minute hour day month weekday |
| Step | `*/15 * * * *` | Every 15 minutes |
| Range | `0 9 * * 1-5` | Monday through Friday at 9 AM |
| List | `0 0 * * 1,3,5` | Monday, Wednesday, Friday at midnight |
| `@yearly` | - | January 1 at 00:00 |
| `@monthly` | - | 1st of each month at 00:00 |
| `@weekly` | - | Sunday at 00:00 |
| `@daily` | - | Every day at 00:00 |
| `@hourly` | - | Every hour at :00 |
| `@every` | `@every 30s` | Fixed interval (minimum 30 seconds) |

***

©️ 2025 [邱敬幃 Pardn Chiu](https://linkedin.com/in/pardnchiu)
