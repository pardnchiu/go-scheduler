# go-scheduler - 技術文件

> 返回 [README](./README.zh.md)

## 前置需求

- Go 1.23 或以上版本
- 非 Windows 平台（logger 依賴 `log/syslog`；syslog 無法連線時自動改寫 stderr）

## 安裝

### 使用 go get

```bash
go get github.com/pardnchiu/go-scheduler@latest
```

套件位於 `core` 子目錄，匯入路徑為：

```go
import "github.com/pardnchiu/go-scheduler/core"
```

### 從原始碼

```bash
git clone https://github.com/pardnchiu/go-scheduler.git
cd go-scheduler
go build ./...
go test -race ./...
```

## 使用方式

### 基礎

建立排程器、加入任務並啟動：

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

	// 等待進行中任務結束
	<-c.Stop().Done()
}
```

### 自訂時區

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

### Cron 表達式

標準五欄位（分 時 日 月 週）：

```go
specs := map[string]func() error{
	"0 * * * *":       doHourlyJob, // 每小時第 0 分
	"30 9 * * 1-5":    sendReport,  // 週一至週五 09:30
	"*/5 * * * *":     poll,        // 分鐘數可被 5 整除時
	"0 9,12,18 * * *": checkpoint,  // 列表
}

for spec, action := range specs {
	if _, err := c.Add(spec, action); err != nil {
		log.Fatalf("add %q: %v", spec, err)
	}
}
```

### 描述符與固定間隔

```go
for _, spec := range []string{
	"@hourly", "@daily", "@midnight", "@weekly",
	"@monthly", "@yearly", "@annually",
	"@every 30s", "@every 5m", "@every 1h", // @every 最小間隔為 30s
} {
	if _, err := c.Add(spec, func() error { return nil }); err != nil {
		log.Fatalf("add %q: %v", spec, err)
	}
}
```

### 任務超時

傳入 `time.Duration` 作為執行時限；另傳入 `func()` 作為逾時回呼：

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

逾時後任務標記為 `TaskFailed`，但 action 所在的 goroutine 不會被中斷，會持續執行到返回為止。

### 任務依賴

有依賴的任務必須使用 `func() error`，並以 `[]core.Wait` 宣告前置任務：

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

依賴行為：

| 情境 | 結果 |
|------|------|
| 所有前置任務狀態為 `TaskCompleted` | 執行依賴任務 |
| 前置任務失敗且 `State: core.Stop` | 依賴任務標記為 `TaskFailed` |
| 前置任務失敗且 `State: core.Skip` | 忽略該前置任務，繼續等待其餘前置任務 |
| 前置任務 ID 不存在 | 依賴任務標記為 `TaskFailed` |
| 1 分鐘內前置任務未完成 | 依賴任務標記為 `TaskFailed` |

- 依賴判斷依據前置任務「目前狀態」，而非同一輪的執行結果
- 以 `func()` 註冊的任務在加入時即為 `TaskCompleted`，依賴它的任務不需等待其首次執行
- 以 `func()` 作為 action 並帶入依賴，`Add` 回傳錯誤

### 進階：移除與列出

```go
// 以 ID 移除
c.Remove(id)

// 移除全部任務
c.RemoveAll()

// 列出目前啟用中的任務
for _, t := range c.List() {
	log.Println(t.ID)
}

// 優雅關閉，最多等待 30 秒
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

select {
case <-c.Stop().Done():
case <-ctx.Done():
	log.Println("shutdown timeout")
}
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
| `Location` | 排程時區；為 `nil` 時使用 `time.Local` |

### New

```go
func New(c Config) (*cron, error)
```

建立排程器實例，初始化最小堆、解析器、依賴子系統與 logger（優先使用 syslog，tag 為 `goCron`；失敗時改寫 stderr）。

### Start / Stop

```go
func (c *cron) Start()
func (c *cron) Stop() context.Context
```

| 方法 | 說明 |
|------|------|
| `Start` | 啟動事件迴圈與依賴 Worker 池；已啟動時為 no-op |
| `Stop` | 停止事件迴圈與 Worker 池，回傳在進行中無依賴任務結束後 cancel 的 `context.Context` |

### Add

```go
func (c *cron) Add(spec string, action interface{}, arg ...interface{}) (int64, error)
```

加入排程任務，回傳遞增的任務 ID。可於 `Start` 前後呼叫。

| 參數 | 型別 | 說明 |
|------|------|------|
| `spec` | `string` | Cron 表達式、描述符或 `@every <duration>` |
| `action` | `func()` 或 `func() error` | 任務本體；有依賴時必須為 `func() error` |
| `arg` | 可變參數 | 見下表，可任意組合 |

| 型別 | 用途 |
|------|------|
| `string` | 任務描述 |
| `time.Duration` | 執行時限 |
| `func()` | 逾時回呼 |
| `[]Wait` | 前置任務 |
| `[]int64` | （已棄用）前置任務 ID 列表，等同 `Wait{ID: id}` |

錯誤：

| 條件 | 錯誤訊息 |
|------|----------|
| `spec` 無法解析 | `failed to parse: ...` |
| `action` 型別不符 | `action need to be func() or func()` |
| `func()` 搭配依賴 | `need return value to get dependence support` |

### Remove / RemoveAll / List

```go
func (c *cron) Remove(id int64)
func (c *cron) RemoveAll()
func (c *cron) List() []*task
```

| 方法 | 說明 |
|------|------|
| `Remove` | 停用並自最小堆移除指定 ID 的任務 |
| `RemoveAll` | 執行中時清空最小堆；未啟動時停用全部任務 |
| `List` | 回傳目前啟用中任務的指標，可讀取 `ID` 欄位 |

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
| `Delay` | 目前未生效；等待前置任務的時限固定為 1 分鐘 |
| `State` | 前置任務失敗時的策略：`Stop` 或 `Skip`（預設 `Stop`） |

### 任務狀態

```go
const (
	TaskPending int = iota
	TaskRunning
	TaskCompleted
	TaskFailed
)
```

### 排程語法

| 格式 | 範例 | 說明 |
|------|------|------|
| 五欄位 Cron | `30 9-17 * * 1-5` | 分（0-59）時（0-23）日（1-31）月（1-12）週（0-6，0 為週日） |
| 描述符 | `@yearly` `@annually` `@monthly` `@weekly` `@daily` `@midnight` `@hourly` | 內建簡寫 |
| 固定間隔 | `@every 30s` | 以 `time.ParseDuration` 解析，最小 30 秒 |
| 欄位語法 | `*` `n` `n-m` `a,b,n-m` `*/n` | 全部、單值、範圍、列表、步進 |

`*/n` 比對「欄位值可被 n 整除」，不支援 `n-m/s` 形式。

***

©️ 2025 [邱敬幃 Pardn Chiu](https://www.linkedin.com/in/pardnchiu)
