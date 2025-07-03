> [!Note]
> This content is translated by LLM. Original text can be found [here](README.zh.md)

# Cron Job Scheduler (Golang)

> A minimal Golang scheduler supporting standard cron expressions, custom descriptors, and custom intervals for easy scheduling in Go.<br>
> Originally designed for the scheduling functionality used in [pardnchiu/go-ip-sentry](https://github.com/pardnchiu/go-ip-sentry) threat score decay calculations.

[![license](https://img.shields.io/github/license/pardnchiu/go-cron)](LICENSE)
[![version](https://img.shields.io/github/v/tag/pardnchiu/go-cron)](https://github.com/pardnchiu/go-cron/releases)
[![readme](https://img.shields.io/badge/readme-中文-blue)](README.zh.md) 

## Three Core Features

### Flexible Syntax
Supports standard cron expressions, custom descriptors (`@hourly`, `@daily`, `@weekly`, etc.), and custom interval (`@every`) syntax

### Concurrent Execution
Concurrent task execution and management with panic recovery mechanism and dynamic task add/remove functionality

### High-Performance Architecture
Min-heap based task scheduling algorithm ensuring optimal performance in high-volume task scenarios

## Flowchart

<details>
<summary>Click to view</summary>

```mermaid
flowchart TD
  A[Initialize] --> C[Setup Logging System]
  C --> D[Initialize Task Heap]
  D --> H{Already Running?}
  H -->|Yes| I[No Action]
  H -->|No| J[Start Execution]
  
  J --> K[Calculate Initial Task Times]
  K --> L[Initialize Min Heap]
  L --> M[Start Main Loop]
  
  M --> N{Check Heap Status}
  N -->|Empty| O[Wait for Events]
  N -->|Has Tasks| P[Set Timer to Next Task]
  
  O --> Q[Listen for Events]
  P --> Q
  
  Q --> R{Event Type}
  R -->|Timer Expired| S[Execute Due Tasks]
  R -->|Add Task| T[Add to Heap]
  R -->|Remove Task| U[Remove from Heap]
  R -->|Stop Signal| V[Cleanup and Exit]
  
  S --> W[Pop Task from Heap]
  W --> X[Check if Enabled]
  X -->|Disabled| Y[Skip Task]
  X -->|Enabled| BB[Execute Task Function]
  BB --> DD[Calculate Next Execution Time]
  BB -->|Panic| CC[Recover]
  CC --> DD[Calculate Next Execution Time]
  DD --> EE[Re-add to Heap if Recurring]
  
  T --> FF[Parse Schedule]
  FF --> GG[Create Task Object]
  GG --> HH[Add to Heap]
  
  U --> II[Find Task by ID]
  II --> JJ[Mark as Disabled]
  JJ --> KK[Remove from Heap]
  
  Y --> M
  EE --> M
  HH --> M
  KK --> M
  
  V --> LL[Wait for Running Tasks to Complete]
  LL --> MM[Close Channels]
  MM --> NN[Scheduler Stopped]
```

</details>

## Dependencies

- [`github.com/pardnchiu/go-logger`](https://github.com/pardnchiu/go-logger)

## Usage

### Installation
```bash
go get github.com/pardnchiu/go-cron
```

### Initialization
```go
package main

import (
  "fmt"
  "log"
  "time"
  
  cj "github.com/pardnchiu/go-cron"
)

func main() {
  config := cj.Config{
    Log: &cj.Log{
      Stdout: true,
    },
    Location: time.Local,
  }
  
  // Initialize
  scheduler, err := cj.New(config)
  if err != nil {
    log.Fatal(err)
  }
  
  // Add tasks with different schedules
  
  // Every 5 minutes
  id1, err := scheduler.Add("*/5 * * * *", func() {
    fmt.Println("Task executed every 5 minutes")
  })
  
  // Every hour
  id2, err := scheduler.Add("@hourly", func() {
    fmt.Println("Hourly task executed")
  })
  
  // Every 30 seconds
  id3, err := scheduler.Add("@every 30s", func() {
    fmt.Println("Task executed every 30 seconds")
  })
  
  if err != nil {
    log.Printf("Failed to add task: %v", err)
  }
  
  time.Sleep(10 * time.Minute)
  
  // Remove task
  scheduler.Remove(id1)
  
  // Stop and wait for completion
  ctx := scheduler.Stop()
  <-ctx.Done()
  
  fmt.Println("Scheduler stopped gracefully")
}
```

## Configuration

```go
type Config struct {
  Log      *Log           // Logging configuration
  Location *time.Location // Timezone setting (default: time.Local)
}

type Log struct {
  Path      string // Log file path (default: ./logs/cron.log)
  Stdout    bool   // Output to stdout (default: false)
  MaxSize   int64  // Maximum log file size in bytes (default: 16MB)
  MaxBackup int    // Number of backup files to retain (default: 5)
  Type      string // Output format: "json" for slog standard, "text" for tree format (default: "text")
}
```

## Supported Formats

### Standard Cron
5-field format: `minute hour day month weekday`

```go
// Every minute
scheduler.Add("* * * * *", task)

// Daily at midnight
scheduler.Add("0 0 * * *", task)

// Weekdays at 9 AM
scheduler.Add("0 9 * * 1-5", task)

// Every 15 minutes
scheduler.Add("*/15 * * * *", task)

// First day of month at 6 AM
scheduler.Add("0 6 1 * *", task)
```

### Custom Descriptors

```go
// January 1st at midnight
scheduler.Add("@yearly", task)

// First day of month at midnight
scheduler.Add("@monthly", task)

// Sunday at midnight
scheduler.Add("@weekly", task)

// Daily at midnight
scheduler.Add("@daily", task)

// Every hour on the hour
scheduler.Add("@hourly", task)

// Every 30 seconds
scheduler.Add("@every 30s", task)

// Every 5 minutes
scheduler.Add("@every 5m", task)

// Every 2 hours
scheduler.Add("@every 2h", task)

// Every 12 hours
scheduler.Add("@every 12h", task)
```

## Available Functions

### Scheduler Management

- **New** - Create a new scheduler instance
  ```go
  scheduler, err := cj.New(config)
  ```
  - Sets up task heap and communication channels
  
- **Start** - Start the scheduler instance
  ```go
  scheduler.Start()
  ```
  - Starts the scheduling loop

- **Stop** - Gracefully stop the scheduler
  ```go
  ctx := scheduler.Stop()
  <-ctx.Done() // Wait for all tasks to complete
  ```
  - Sends stop signal to main loop
  - Returns context that completes when all running tasks finish
  - Ensures graceful shutdown without interrupting tasks

### Task Management

- **Add** - Add a scheduled task
  ```go
  taskID, err := scheduler.Add("0 */2 * * *", func() {
    // Task logic
  })
  ```
  - Parses schedule syntax
  - Generates unique task ID for management

- **Remove** - Cancel a scheduled task
  ```go
  scheduler.Remove(taskID)
  ```
  - Removes task from scheduling queue
  - Safe to call regardless of scheduler state

## Upcoming Features
- Import task dependency management as in [php-async](https://github.com/pardnchiu/php-async)
  - Pre-dependencies: Task B executes after Task A completes
  - Post-dependencies: Task B executes before Task A starts
  - Multiple dependencies: Task C waits for both Tasks A and B to complete

## License

This project is licensed under the [MIT](LICENSE) License.

## Author

<img src="https://avatars.githubusercontent.com/u/25631760" align="left" width="96" height="96" style="margin-right: 0.5rem;">

<h4 style="padding-top: 0">邱敬幃 Pardn Chiu</h4>

<a href="mailto:dev@pardn.io" target="_blank">
  <img src="https://pardn.io/image/email.svg" width="48" height="48">
</a> <a href="https://linkedin.com/in/pardnchiu" target="_blank">
  <img src="https://pardn.io/image/linkedin.svg" width="48" height="48">
</a>

***

©️ 2025 [邱敬幃 Pardn Chiu](https://pardn.io)