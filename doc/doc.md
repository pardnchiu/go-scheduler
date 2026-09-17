# go-scheduler - Documentation

> Back to [README](../README.md)

## Prerequisites

- Go 1.23 or higher
- A non-Windows platform (the logger depends on `log/syslog`; it falls back to stderr when syslog is unreachable)

## Installation

### Using go get

```bash
go get github.com/pardnchiu/go-scheduler@latest
```

The package lives in the `core` subdirectory:

```go
import "github.com/pardnchiu/go-scheduler/core"
```

### From Source

```bash
git clone https://github.com/pardnchiu/go-scheduler.git
cd go-scheduler
go build ./...
go test -race ./...
```

## Usage

### Basic

Create a scheduler, add a task, and start it:

```go
package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/pardnchiu/go-scheduler/core"
)

func main() {
	c, err := core.New(core.Config{})
	if err != nil {
		log.Fatal(err)
	}

	id, err := c.Add("@every 30s", func() {
		log.Println("heartbeat")
	}, "heartbeat")
	if err != nil {
		log.Fatal(err)
	}
	log.Println("task id:", id)

	c.Start()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	// wait for in-flight tasks
	<-c.Stop().Done()
}
```

### Custom Timezone

```go
loc, err := time.LoadLocation("Asia/Taipei")
if err != nil {
	log.Fatal(err)
}

c, err := core.New(core.Config{Location: loc})
if err != nil {
	log.Fatal(err)
}
```

### Cron Expressions

Standard 5 fields (minute hour day month weekday):

```go
specs := map[string]func() error{
	"0 * * * *":       doHourlyJob, // minute 0 of every hour
	"30 9 * * 1-5":    sendReport,  // 09:30 Monday to Friday
	"*/5 * * * *":     poll,        // minutes divisible by 5
	"0 9,12,18 * * *": checkpoint,  // list
}

for spec, action := range specs {
	if _, err := c.Add(spec, action); err != nil {
		log.Fatalf("add %q: %v", spec, err)
	}
}
```

### Descriptors and Fixed Intervals

```go
for _, spec := range []string{
	"@hourly", "@daily", "@midnight", "@weekly",
	"@monthly", "@yearly", "@annually",
	"@every 30s", "@every 5m", "@every 1h", // @every minimum interval is 30s
} {
	if _, err := c.Add(spec, func() error { return nil }); err != nil {
		log.Fatalf("add %q: %v", spec, err)
	}
}
```

### Task Timeout

Pass a `time.Duration` as the execution limit and a `func()` as the timeout callback:

```go
_, err := c.Add("@every 1m", func() error {
	time.Sleep(10 * time.Second)
	return nil
}, "slow job", 3*time.Second, func() {
	log.Println("task timed out")
})
if err != nil {
	log.Fatal(err)
}
```

A timed-out task is marked `TaskFailed`, but the goroutine running the action is not interrupted and keeps running until it returns.

### Task Dependencies

Dependent tasks must use `func() error` and declare prerequisites with `[]core.Wait`:

```go
prepareID, err := c.Add("@every 1m", func() error {
	return prepare()
}, "prepare")
if err != nil {
	log.Fatal(err)
}

_, err = c.Add("@every 1m", func() error {
	return process()
}, "process", []core.Wait{
	{ID: prepareID, State: core.Stop},
})
if err != nil {
	log.Fatal(err)
}
```

Dependency behavior:

| Condition | Result |
|-----------|--------|
| Every prerequisite is `TaskCompleted` | The dependent task runs |
| A prerequisite failed with `State: core.Stop` | The dependent task is marked `TaskFailed` |
| A prerequisite failed with `State: core.Skip` | That prerequisite is ignored; the rest are still awaited |
| A prerequisite ID does not exist | The dependent task is marked `TaskFailed` |
| Prerequisites are not done within 1 minute | The dependent task is marked `TaskFailed` |

- Dependencies are evaluated against each prerequisite's current state, not against a run in the same cycle
- Tasks registered with `func()` start as `TaskCompleted`, so their dependents do not wait for their first run
- `Add` returns an error when a `func()` action is combined with dependencies

### Advanced: Remove and List

```go
// remove by ID
c.Remove(id)

// remove every task
c.RemoveAll()

// list currently enabled tasks
for _, t := range c.List() {
	log.Println(t.ID)
}

// graceful shutdown, waiting at most 30 seconds
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

select {
case <-c.Stop().Done():
case <-ctx.Done():
	log.Println("shutdown timeout")
}
```

## API Reference

### Config

```go
type Config struct {
	Location *time.Location
}
```

| Field | Description |
|-------|-------------|
| `Location` | Schedule timezone; `time.Local` when `nil` |

### New

```go
func New(c Config) (*cron, error)
```

Creates a scheduler and initializes the min-heap, parser, dependency subsystem, and logger (syslog with tag `goCron` first, stderr on failure).

### Start / Stop

```go
func (c *cron) Start()
func (c *cron) Stop() context.Context
```

| Method | Description |
|--------|-------------|
| `Start` | Starts the event loop and the dependency worker pool; no-op when already running |
| `Stop` | Stops the event loop and worker pool; returns a `context.Context` cancelled after in-flight tasks without dependencies finish |

### Add

```go
func (c *cron) Add(spec string, action interface{}, arg ...interface{}) (int64, error)
```

Adds a scheduled task and returns a monotonically increasing ID. Callable before or after `Start`.

| Parameter | Type | Description |
|-----------|------|-------------|
| `spec` | `string` | Cron expression, descriptor, or `@every <duration>` |
| `action` | `func()` or `func() error` | Task body; must be `func() error` when dependencies are set |
| `arg` | variadic | Any combination of the types below |

| Type | Purpose |
|------|---------|
| `string` | Task description |
| `time.Duration` | Execution limit |
| `func()` | Timeout callback |
| `[]Wait` | Prerequisites |
| `[]int64` | (Deprecated) prerequisite IDs, equivalent to `Wait{ID: id}` |

Errors:

| Condition | Message |
|-----------|---------|
| `spec` cannot be parsed | `failed to parse: ...` |
| Unsupported `action` type | `action need to be func() or func()` |
| `func()` with dependencies | `need return value to get dependence support` |

### Remove / RemoveAll / List

```go
func (c *cron) Remove(id int64)
func (c *cron) RemoveAll()
func (c *cron) List() []*task
```

| Method | Description |
|--------|-------------|
| `Remove` | Disables the task with the given ID and removes it from the heap |
| `RemoveAll` | Clears the heap while running; disables every task before `Start` |
| `List` | Returns pointers to enabled tasks; the `ID` field is readable |

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
|-------|-------------|
| `ID` | Prerequisite task ID |
| `Delay` | Currently not applied; the prerequisite wait limit is fixed at 1 minute |
| `State` | Policy when the prerequisite fails: `Stop` or `Skip` (default `Stop`) |

### Task States

```go
const (
	TaskPending int = iota
	TaskRunning
	TaskCompleted
	TaskFailed
)
```

### Schedule Syntax

| Format | Example | Description |
|--------|---------|-------------|
| 5-field cron | `30 9-17 * * 1-5` | minute (0-59) hour (0-23) day (1-31) month (1-12) weekday (0-6, 0 is Sunday) |
| Descriptors | `@yearly` `@annually` `@monthly` `@weekly` `@daily` `@midnight` `@hourly` | Built-in shortcuts |
| Fixed interval | `@every 30s` | Parsed by `time.ParseDuration`, minimum 30 seconds |
| Field syntax | `*` `n` `n-m` `a,b,n-m` `*/n` | all, single, range, list, step |

`*/n` matches field values divisible by n; `n-m/s` is not supported.

***

©️ 2025 [邱敬幃 Pardn Chiu](https://www.linkedin.com/in/pardnchiu)
