# go-scheduler - Documentation

> Back to [README](../README.md)

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

Create a scheduler, add a task, and start it:

```go
package main

import (
	"fmt"
	"time"

	"github.com/pardnchiu/go-scheduler/core"
)

func main() {
	c, err := core.New(core.Config{})
	if err != nil {
		panic(err)
	}

	id, err := c.Add("@every 30s", func() {
		fmt.Println("hello", time.Now())
	}, "heartbeat")
	if err != nil {
		panic(err)
	}
	fmt.Println("task id:", id)

	c.Start()
	defer func() {
		ctx := c.Stop()
		<-ctx.Done()
	}()

	time.Sleep(2 * time.Minute)
}
```

### Custom Timezone

```go
loc, err := time.LoadLocation("Asia/Taipei")
if err != nil {
	panic(err)
}

c, err := core.New(core.Config{
	Location: loc,
})
if err != nil {
	panic(err)
}
```

### Cron Expressions

Standard 5-field expressions (minute hour day month weekday):

```go
// minute 0 of every hour
c.Add("0 * * * *", func() error {
	return doHourlyJob()
})

// 09:30 on weekdays
c.Add("30 9 * * 1-5", func() error {
	return sendReport()
})

// every 5 minutes
c.Add("*/5 * * * *", func() error {
	return poll()
})

// lists and ranges
c.Add("0 9,12,18 * * *", func() error {
	return checkpoint()
})
```

### Descriptors and Fixed Intervals

```go
c.Add("@hourly", func() error { return nil })
c.Add("@daily", func() error { return nil })
c.Add("@weekly", func() error { return nil })
c.Add("@monthly", func() error { return nil })
c.Add("@yearly", func() error { return nil })

// @every minimum interval is 30s
c.Add("@every 30s", func() error { return nil })
c.Add("@every 5m", func() error { return nil })
c.Add("@every 1h", func() error { return nil })
```

### Task Timeout

Pass a `time.Duration` as the execution timeout; optionally pass a timeout callback:

```go
c.Add("@every 1m", func() error {
	time.Sleep(10 * time.Second)
	return nil
}, 3*time.Second, func() {
	fmt.Println("task timed out")
})
```

### Task Dependencies

Dependent tasks must use `func() error` and declare prerequisites with `[]core.Wait`:

```go
parentID, err := c.Add("@every 1m", func() error {
	return prepare()
}, "prepare")
if err != nil {
	panic(err)
}

childID, err := c.Add("@every 1m", func() error {
	return process()
}, "process", []core.Wait{
	{ID: parentID, Delay: 10 * time.Second, State: core.Stop},
})
if err != nil {
	panic(err)
}

_ = childID
```

`Wait.State`:

- `core.Stop`: fail and stop the dependent task when a prerequisite fails
- `core.Skip`: skip the failed prerequisite and keep waiting for the rest

### Advanced: Remove and List

```go
// remove by ID
c.Remove(id)

// clear all tasks
c.RemoveAll()

// list currently enabled tasks
tasks := c.List()
for _, t := range tasks {
	fmt.Println(t.ID)
}

// graceful stop: wait for in-flight tasks
ctx := c.Stop()
<-ctx.Done()
```

## API Reference

### Config

```go
type Config struct {
	Location *time.Location
}
```

| Field | Description |
|------|------|
| `Location` | Schedule timezone; uses `time.Local` when `nil` |

### New

```go
func New(c Config) (*cron, error)
```

Creates a scheduler instance. Initializes the min-heap, parser, dependency manager, and logger (prefers syslog, falls back to stderr).

### Start / Stop

```go
func (c *cron) Start()
func (c *cron) Stop() context.Context
```

- `Start`: starts the main loop and dependency worker pool; safe to call repeatedly (no-op if already running)
- `Stop`: stops the scheduler and workers, returns a `context.Context` that cancels after in-flight tasks finish

### Add

```go
func (c *cron) Add(spec string, action interface{}, arg ...interface{}) (int64, error)
```

Adds a scheduled task and returns a monotonic task ID.

| Parameter | Type | Description |
|------|------|------|
| `spec` | `string` | Cron expression, descriptor, or `@every <duration>` |
| `action` | `func()` or `func() error` | Task body; dependencies require `func() error` |
| `arg` | variadic | See optional arguments below |

Optional arguments (any combination):

| Type | Purpose |
|------|------|
| `string` | Task description |
| `time.Duration` | Execution timeout |
| `func()` | Timeout callback (`onDelay`) |
| `[]Wait` | Prerequisite dependencies |
| `[]int64` | (Deprecated) prerequisite task ID list |

### Remove / RemoveAll / List

```go
func (c *cron) Remove(id int64)
func (c *cron) RemoveAll()
func (c *cron) List() []task
```

| Method | Description |
|------|------|
| `Remove` | Disable and remove a task by ID |
| `RemoveAll` | Clear all tasks from the heap |
| `List` | Return copies of currently enabled tasks |

### Wait / WaitState

```go
type Wait struct {
	ID    int64
	Delay time.Duration
	State WaitState
}

type WaitState int

const (
	Stop WaitState = iota
	Skip
)
```

| Field | Description |
|------|------|
| `ID` | Prerequisite task ID |
| `Delay` | Timeout waiting for prerequisites; dependency worker defaults to 1 minute when `0` |
| `State` | Failure policy for prerequisites: `Stop` or `Skip` |

### Task States

```go
const (
	TaskPending int = iota
	TaskRunning
	TaskCompleted
	TaskFailed
)
```

### Schedule Syntax Summary

| Format | Example | Description |
|------|------|------|
| 5-field cron | `*/5 9-17 * * 1-5` | minute hour day month weekday |
| Descriptors | `@hourly` `@daily` `@weekly` `@monthly` `@yearly` | Built-in shortcuts |
| Fixed interval | `@every 30s` | Minimum 30 seconds |
| Field syntax | `*` `n` `n-m` `a,b,c` `*/n` | all, single, range, list, step |

***

©️ 2025 [邱敬幃 Pardn Chiu](https://linkedin.com/in/pardnchiu)
