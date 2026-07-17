# go-scheduler - 技術文件

> 返回 [README](./README.zh.md)

## 前置需求

- Go 1.23 或更高版本

## 安裝

### 使用 go get

```bash
go get github.com/pardnchiu/go-scheduler
```

### 從原始碼建置

```bash
git clone https://github.com/pardnchiu/go-scheduler.git
cd go-scheduler
go build ./...
```

## 使用方式

### 基礎用法

建立排程器、新增任務並啟動：

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

### 指定時區

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

### Cron 表達式

標準五欄位表達式（分 時 日 月 週）：

```go
// 每小時第 0 分
c.Add("0 * * * *", func() error {
	return doHourlyJob()
})

// 工作日早上 9:30
c.Add("30 9 * * 1-5", func() error {
	return sendReport()
})

// 每 5 分鐘
c.Add("*/5 * * * *", func() error {
	return poll()
})

// 列表與範圍
c.Add("0 9,12,18 * * *", func() error {
	return checkpoint()
})
```

### 描述符與固定間隔

```go
c.Add("@hourly", func() error { return nil })
c.Add("@daily", func() error { return nil })
c.Add("@weekly", func() error { return nil })
c.Add("@monthly", func() error { return nil })
c.Add("@yearly", func() error { return nil })

// @every 最小間隔 30s
c.Add("@every 30s", func() error { return nil })
c.Add("@every 5m", func() error { return nil })
c.Add("@every 1h", func() error { return nil })
```

### 任務超時

傳入 `time.Duration` 作為執行逾時；可再傳逾時回呼：

```go
c.Add("@every 1m", func() error {
	time.Sleep(10 * time.Second)
	return nil
}, 3*time.Second, func() {
	fmt.Println("task timed out")
})
```

### 任務依賴

依賴任務必須使用 `func() error`，並以 `[]core.Wait` 宣告前置任務：

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

`Wait.State`：

- `core.Stop`：前置任務失敗時，後續任務失敗並停止
- `core.Skip`：前置任務失敗時略過該依賴，繼續等待其餘依賴

### 進階：移除與列表

```go
// 依 ID 移除
c.Remove(id)

// 清空全部
c.RemoveAll()

// 列出目前啟用中的任務
tasks := c.List()
for _, t := range tasks {
	fmt.Println(t.ID)
}

// 優雅停止：等待執行中任務結束
ctx := c.Stop()
<-ctx.Done()
```

## API 參考

### Config

```go
type Config struct {
	Location *time.Location
}
```

| 欄位 | 說明 |
|------|------|
| `Location` | 排程時區；`nil` 時使用 `time.Local` |

### New

```go
func New(c Config) (*cron, error)
```

建立排程實例。初始化最小堆、解析器、依賴管理與 logger（優先 syslog，失敗則回退 stderr）。

### Start / Stop

```go
func (c *cron) Start()
func (c *cron) Stop() context.Context
```

- `Start`：啟動主迴圈與依賴 worker 池；重複呼叫安全（已啟動則略過）
- `Stop`：停止排程與 worker，回傳 `context.Context`；於所有執行中任務結束後取消

### Add

```go
func (c *cron) Add(spec string, action interface{}, arg ...interface{}) (int64, error)
```

新增排程任務，回傳遞增的任務 ID。

| 參數 | 型別 | 說明 |
|------|------|------|
| `spec` | `string` | Cron 表達式、描述符或 `@every <duration>` |
| `action` | `func()` 或 `func() error` | 任務本體；依賴必須用 `func() error` |
| `arg` | variadic | 見下方選用參數 |

選用參數（可任意組合）：

| 型別 | 用途 |
|------|------|
| `string` | 任務描述 |
| `time.Duration` | 執行逾時 |
| `func()` | 逾時回呼（`onDelay`） |
| `[]Wait` | 前置依賴 |
| `[]int64` | （已棄用）前置任務 ID 列表 |

### Remove / RemoveAll / List

```go
func (c *cron) Remove(id int64)
func (c *cron) RemoveAll()
func (c *cron) List() []task
```

| 方法 | 說明 |
|------|------|
| `Remove` | 停用並移除指定任務 |
| `RemoveAll` | 清空 heap 內全部任務 |
| `List` | 回傳目前啟用中任務的副本 |

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

| 欄位 | 說明 |
|------|------|
| `ID` | 前置任務 ID |
| `Delay` | 等待前置完成的逾時；`0` 時依賴 worker 預設 1 分鐘 |
| `State` | 前置失敗策略：`Stop` 或 `Skip` |

### 任務狀態

```go
const (
	TaskPending int = iota
	TaskRunning
	TaskCompleted
	TaskFailed
)
```

### 排程語法摘要

| 格式 | 範例 | 說明 |
|------|------|------|
| 五欄位 cron | `*/5 9-17 * * 1-5` | 分 時 日 月 週 |
| 描述符 | `@hourly` `@daily` `@weekly` `@monthly` `@yearly` | 內建捷徑 |
| 固定間隔 | `@every 30s` | 最小 30 秒 |
| 欄位語法 | `*` `n` `n-m` `a,b,c` `*/n` | 全選、單值、範圍、列表、步進 |

***

©️ 2025 [邱敬幃 Pardn Chiu](https://linkedin.com/in/pardnchiu)
