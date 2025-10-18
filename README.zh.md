# Go 定時排程

> 輕量的 Golang 排程器，支援標準 cron 表達式、自定義描述符、自訂間隔和任務依賴關係。輕鬆使用 Go 撰寫排程<br>
> 原本是設計給 [pardnchiu/go-ip-sentry](https://github.com/pardnchiu/go-ip-sentry) 威脅分數衰退計算所使用到的排程功能

[![pkg](https://pkg.go.dev/badge/github.com/pardnchiu/go-scheduler.svg)](https://pkg.go.dev/github.com/pardnchiu/go-scheduler)
[![card](https://goreportcard.com/badge/github.com/pardnchiu/go-scheduler)](https://goreportcard.com/report/github.com/pardnchiu/go-scheduler)
[![codecov](https://img.shields.io/codecov/c/github/pardnchiu/go-scheduler)](https://app.codecov.io/github/pardnchiu/go-scheduler)
[![version](https://img.shields.io/github/v/tag/pardnchiu/go-scheduler?label=release)](https://github.com/pardnchiu/go-scheduler/releases)
[![license](https://img.shields.io/github/license/pardnchiu/go-scheduler)](LICENSE)
[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge.svg)](https://github.com/avelino/awesome-go)<br>
[![readme](https://img.shields.io/badge/readme-EN-white)](README.md)
[![readme](https://img.shields.io/badge/readme-ZH-white)](README.zh.md)

- [三大核心特色](#三大核心特色)
  - [靈活語法](#靈活語法)
  - [任務依賴](#任務依賴)
  - [高效架構](#高效架構)
- [流程圖](#流程圖)
- [依賴套件](#依賴套件)
- [使用方法](#使用方法)
  - [安裝](#安裝)
  - [初始化](#初始化)
    - [基本使用](#基本使用)
    - [任務依賴](#任務依賴-1)
- [配置介紹](#配置介紹)
- [支援格式](#支援格式)
  - [標準](#標準)
  - [自定義](#自定義)
- [可用函式](#可用函式)
  - [排程管理](#排程管理)
  - [任務管理](#任務管理)
- [任務依賴](#任務依賴-2)
  - [基本使用](#基本使用-1)
  - [依賴範例](#依賴範例)
  - [任務狀態](#任務狀態)
- [超時機制](#超時機制)
  - [特點](#特點)
- [功能預告](#功能預告)
  - [任務依賴增強](#任務依賴增強)
  - [任務完成觸發改寫](#任務完成觸發改寫)
- [授權條款](#授權條款)
- [星](#星)
- [作者](#作者)

## 三大核心特色

### 靈活語法
支援標準 cron 表達式、自定義描述符（`@hourly`、`@daily`、`@weekly` 等）和自訂間隔（`@every`）語法，零學習成本，只要會寫 cron 表達式就基本會使用

### 任務依賴
支援前置依賴任務、多重依賴、依賴超時控制和失敗處理機制

### 高效架構
使用 Golang 標準庫的 `heap`，專注核心功能，基於最小堆的任務排程，併發的任務執行和管理，具有 panic 恢復機制和動態任務新增/移除功能，並確保在大量任務場景中的最佳效能

## 流程圖

<details>
<summary>主流程</summary> 

```mermaid
flowchart TD
  A[初始化] --> B{是否已執行?}
  B -->|否| B0[開始執行]
    B0 --> C[計算初始任務]
    C --> D[初始化任務]
    D --> E[啟動主迴圈]
      E --> H{檢查堆狀態}
      E -->|無任務<br>等待事件| Q
      E -->|有任務<br>設置下一任務計時器| Q
  B -->|是<br>等待觸發| Q[監聽事件]
  
  Q --> R{事件類型}
  R -->|計時器到期| R1[執行到期任務]
  R -->|新增任務| R2[加入堆]
  R -->|移除任務| R3[從堆移除]
  R -->|停止信號| R4[清理並退出]
  
  R1 --> S[從堆中彈出任務]
  S --> R5[計算下一次執行時間]
  R5 --> E
  S --> T{檢查是否啟用}
  T -->|未啟用| T0[跳過任務]
  T0 --> E
  T -->|啟用| T1[執行任務函數]
  
  R2 --> V[解析排程]
  V --> W[建立任務物件]
  W --> X[加入堆]
  
  R3 --> Y[根據 ID 查找任務]
  Y --> Z[標記為未啟用]
  Z --> AA[從堆移除]
  
  X --> E
  AA --> E
  
  R4 --> BB[等待執行中的任務完成]
  BB --> CC[關閉通道]
  CC --> DD[排程器已停止]
```

</details>

<details>
<summary>依賴流程</summary>

```mermaid
flowchart TD
    A[任務加入執行佇列] --> B{檢查依賴}
    B -->|無依賴| B0[跳過依賴流程]
      B0 --> Z[結束]
    B -->|有依賴| B1{依賴完成?}
      B1 -->|否| B10[等待依賴完成]
        B10 --> C{依賴等待超時?}
          C -->|否| C0[繼續等待]
            C0 --> D{依賴解決?}
              D -->|失敗<br>標記失敗| V
              D -->|完成| B11
              D -->|仍在等待| B10
          C -->|是<br>標記失敗| V
      B1 -->|是| B11[執行]
        B11 -->|標記執行中| E{任務超時存在?}
          E -->|否| E0[執行動作]
            E0 --> R{執行結果}
              R -->|成功<br>標記完成| V[更新任務結果]
              R -->|錯誤<br>標記失敗| V
              R -->|Panic<br>恢復並標記失敗| V
          E -->|是| E1{任務超時?}
            E1 -->|超時<br>標記失敗<br>觸發超時動作| V
            E1 -->|未超時| E0
      B1 -->|失敗<br>標記失敗| V
    
    V --> X[記錄執行結果]
    X --> Y[通知依賴任務]
    Y --> Z[結束]
```

</details>

## 依賴套件

- ~~[`github.com/pardnchiu/go-logger`](https://github.com/pardnchiu/go-logger)~~ (< v0.3.1)<br>
  為了效能與穩定度，`v0.3.1` 起棄用非標準庫套件，改用 `log/slog`

## 使用方法

### 安裝

> [!NOTE]
> 最新 commit 可能會變動，建議使用標籤版本<br>
> 針對僅包含文檔更新等非功能改動的 commit，後續會進行 rebase

```bash
go get github.com/pardnchiu/go-scheduler@[VERSION]

git clone --depth 1 --branch [VERSION] https://github.com/pardnchiu/go-scheduler.git
```

### 初始化

#### 基本使用
```go
package main

import (
  "fmt"
  "log"
  "time"
  
  cron "github.com/pardnchiu/go-scheduler"
)

func main() {
  // Initialize (optional configuration)
  scheduler, err := cron.New(cron.Config{
    Location: time.Local,
  })
  if err != nil {
    log.Fatal(err)
  }
  
  // Start scheduler
  scheduler.Start()
  
  // Add tasks
  id1, _ := scheduler.Add("@daily", func() {
    fmt.Println("Daily execution")
  }, "Backup task")
  
  id2, _ := scheduler.Add("@every 5m", func() {
    fmt.Println("Execute every 5 minutes")
  })
  
  // View task list
  tasks := scheduler.List()
  fmt.Printf("Currently have %d tasks\n", len(tasks))
  
  // Remove specific task
  scheduler.Remove(id1)
  
  // Remove all tasks
  scheduler.RemoveAll()
  
  // Graceful shutdown
  ctx := scheduler.Stop()
  <-ctx.Done()
}
```

#### 任務依賴
```go
package main

import (
  "fmt"
  "log"
  "time"
  
  cron "github.com/pardnchiu/go-scheduler"
)

func main() {
  scheduler, err := cron.New(cron.Config{})
  if err != nil {
    log.Fatal(err)
  }
  
  scheduler.Start()
  defer func() {
    ctx := scheduler.Stop()
    <-ctx.Done()
  }()
  
  // Task A: Data preparation
  taskA, _ := scheduler.Add("0 1 * * *", func() error {
    fmt.Println("Preparing data...")
    time.Sleep(2 * time.Second)
    return nil
  }, "Data preparation")
  
  // Task B: Data processing  
  taskB, _ := scheduler.Add("0 2 * * *", func() error {
    fmt.Println("Processing data...")
    time.Sleep(3 * time.Second)
    return nil
  }, "Data processing")
  
  // Task C: Report generation (depends on A and B)
  taskC, _ := scheduler.Add("0 3 * * *", func() error {
    fmt.Println("Generating report...")
    time.Sleep(1 * time.Second)
    return nil
  }, "Report generation", []Wait{{ID: taskA}, {ID: taskB}})
  
  // Task D: Email sending (depends on C)
  _, _ = scheduler.Add("0 4 * * *", func() error {
    fmt.Println("Sending email...")
    return nil
  }, "Email notification", []Wait{{ID: taskC}})
  
  time.Sleep(10 * time.Second)
}
```

## 配置介紹
```go
type Config struct {
  Location *time.Location // Timezone setting (default: time.Local)
}
```

## 支援格式

### 標準
> 5 欄位格式：`分鐘 小時 日 月 星期`<br>
> 支援範圍語法 `1-5` 和 `1,3,5`

```go
// Every minute
scheduler.Add("* * * * *", task)

// Daily at midnight
scheduler.Add("0 0 * * *", task)

// Every 15 minutes
scheduler.Add("*/15 * * * *", task)

// First day of month at 6 AM
scheduler.Add("0 6 1 * *", task)

// Monday to Wednesday, and Friday
scheduler.Add("0 0 * * 1-3,5", task)
```

### 自定義
```go
// January 1st at midnight
scheduler.Add("@yearly", task)

// First day of month at midnight
scheduler.Add("@monthly", task)

// Every Sunday at midnight
scheduler.Add("@weekly", task)

// Daily at midnight
scheduler.Add("@daily", task)

// Every hour on the hour
scheduler.Add("@hourly", task)

// Every 30 seconds (minimum interval: 30 seconds)
scheduler.Add("@every 30s", task)

// Every 5 minutes
scheduler.Add("@every 5m", task)

// Every 2 hours
scheduler.Add("@every 2h", task)

// Every 12 hours
scheduler.Add("@every 12h", task)
```

## 可用函式

### 排程管理

- `New()` - 建立新的排程實例
  ```go
  scheduler, err := cron.New(config)
  ```
  - 設置任務堆和通訊通道

- `Start()` - 啟動排程實例
  ```go
  scheduler.Start()
  ```
  - 啟動排程迴圈

- `Stop()` - 停止排程器
  ```go
  ctx := scheduler.Stop()
  <-ctx.Done() // Wait for all tasks to complete
  ```
  - 向主迴圈發送停止信號
  - 回傳在所有執行中任務完成時完成的 context
  - 確保不中斷任務的關閉

### 任務管理

- `Add()` - 新增排程任務
  ```go
  // Basic usage (no return value)
  taskID, err := scheduler.Add("0 */2 * * *", func() {
    // Task logic
  })

  // Task with error return (supports dependencies)
  taskID, err := scheduler.Add("@daily", func() error {
    // Task logic
    return nil
  }, "Backup task")

  // Task with timeout control
  taskID, err := scheduler.Add("@hourly", func() error {
    // Long-running task
    time.Sleep(10 * time.Second)
    return nil
  }, "Data processing", 5*time.Second)

  // Task with timeout callback
  taskID, err := scheduler.Add("@daily", func() error {
    // Potentially timeout-prone task
    return heavyProcessing()
  }, "Critical backup", 30*time.Second, func() {
    log.Println("Backup task timed out, please check system status")
  })

  // Task with dependencies
  taskID, err := scheduler.Add("@daily", func() error {
    // Task that depends on other tasks
    return processData()
  }, "Data processing", []Wait{{ID: taskA}, {ID: taskB}})

  // Task with dependencies and timeout
  taskID, err := scheduler.Add("@daily", func() error {
    return generateReport()
  }, "Report generation", []Wait{
    {ID: taskA, Delay: 30 * time.Second},
    {ID: taskB, Delay: 45 * time.Second},
  })
  ```
  - 解析排程語法
  - 產生唯一的任務 ID 以便管理
  - 支援可變參數配置
    - `string`：任務描述
    - `time.Duration`：任務執行超時時間
    - `func()`：超時觸發的回調函式
    - `[]Wait`：依賴任務配置（推薦格式）
    - `[]int64`：依賴任務 ID 列表（v2.0 後將移除）
  - 支援兩種動作函式
    - `func()`：無錯誤返回，不支援依賴
    - `func() error`：有錯誤返回，支援依賴

- `Remove()` - 取消任務排程
  ```go
  scheduler.Remove(taskID)
  ```
  - 從排程佇列中移除任務
  - 無論排程器狀態如何都可安全呼叫

- `RemoveAll()` - 移除所有任務
  ```go
  scheduler.RemoveAll()
  ```
  - 立即移除所有排程任務
  - 不影響正在執行的任務

- `List()` - 獲取任務列表
  ```go
  tasks := scheduler.List()
  ```

## 任務依賴 

### 基本使用
- 無依賴：直接執行
- 有依賴：透過 worker 池和依賴管理器執行
  - 單一依賴：任務 B 在任務 A 完成後執行
  - 多重依賴：任務 C 等待任務 A、B 全部完成後執行
  - 依賴任務超時：等待依賴任務完成的最大時間（預設 1 分鐘）

### 依賴範例

**失敗處理策略**：
```go
// Skip：依賴失敗時跳過並繼續執行
taskC, _ := scheduler.Add("0 3 * * *", func() error {
    fmt.Println("Generating report...")
    return nil
}, "Report generation", []Wait{
    {ID: taskA, State: Skip},  // taskA 失敗時跳過
    {ID: taskB, State: Stop},  // taskB 失敗時停止（預設）
})
```

**自定義超時時間**：
```go
// 為每個依賴設定獨立的等待時間
taskC, _ := scheduler.Add("0 3 * * *", func() error {
    fmt.Println("Generating report...")
    return nil
}, "Report generation", []Wait{
    {ID: taskA, Delay: 30 * time.Second},  // 等待 30 秒
    {ID: taskB, Delay: 45 * time.Second},  // 等待 45 秒
})
```

**組合使用**：
```go
// 結合失敗策略和自定義超時
taskC, _ := scheduler.Add("0 3 * * *", func() error {
    fmt.Println("Generating report...")
    return nil
}, "Report generation", []Wait{
    {ID: taskA, Delay: 30 * time.Second, State: Skip},
    {ID: taskB, Delay: 45 * time.Second, State: Stop},
})
```

### 任務狀態
```go
const (
    TaskPending     // Waiting
    TaskRunning     // Running 
    TaskCompleted   // Completed
    TaskFailed      // Failed / Timeout
)
```

## 超時機制
當執行時間超過設定的 `Delay`
- 中斷任務執行
- 觸發 `OnDelay` 函式（如果有設定）
- 記錄超時日誌
- 繼續執行下一個排程

### 特點
- 超時使用 `context.WithTimeout` 實現
- 超時不會影響其他任務的執行
- 如果動作在超時前完成，不會觸發超時

## 功能評估

### 任務依賴增強

- 狀態回調：新增 `OnTimeout` 和 `OnFailed` 回調函數，方便監控和響應依賴任務的異常狀態

### 任務完成觸發改寫

- 事件驅動：將當前的輪詢改為完全基於 `channel` 的模式，降低 CPU 使用率
- 依賴喚醒：實作依賴任務完成時的主動通知機制，消除無效的輪詢檢查

## 授權條款

此專案採用 [MIT](LICENSE) 授權條款。

## 星

[![Star](https://api.star-history.com/svg?repos=pardnchiu/go-scheduler&type=Date)](https://www.star-history.com/#pardnchiu/go-scheduler&Date)

## 作者

<img src="https://avatars.githubusercontent.com/u/25631760" align="left" width="96" height="96" style="margin-right: 0.5rem;">

<h4 style="padding-top: 0">邱敬幃 Pardn Chiu</h4>

<a href="mailto:dev@pardn.io" target="_blank">
  <img src="https://pardn.io/image/email.svg" width="48" height="48">
</a> <a href="https://linkedin.com/in/pardnchiu" target="_blank">
  <img src="https://pardn.io/image/linkedin.svg" width="48" height="48">
</a>

***

©️ 2025 [邱敬幃 Pardn Chiu](https://pardn.io)