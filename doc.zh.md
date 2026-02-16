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

建立排程器並新增週期性任務：

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

	// 每 30 秒執行一次
	_, err = c.Add("@every 30s", func() {
		fmt.Println("每 30 秒執行")
	}, "定時任務")
	if err != nil {
		log.Fatal(err)
	}

	c.Start()
	defer c.Stop()

	// 保持主程式運行
	select {}
}
```

### 使用標準 Cron 語法

```go
// 每天凌晨 2:30 執行
c.Add("30 2 * * *", func() {
	fmt.Println("每日備份")
}, "每日備份任務")

// 每週一上午 9 點執行
c.Add("0 9 * * 1", func() {
	fmt.Println("週報")
}, "週報任務")

// 每 15 分鐘執行一次
c.Add("*/15 * * * *", func() {
	fmt.Println("健康檢查")
}, "健康檢查")
```

### 帶有超時控制的任務

```go
import "time"

id, err := c.Add("@every 1m", func() error {
	// 長時間運行的任務
	time.Sleep(10 * time.Second)
	return nil
}, "超時任務", 5*time.Second, func() {
	// 超時時觸發的 callback
	fmt.Println("任務超時，執行清理")
})
```

### 任務依賴鏈

```go
// 任務 1：無依賴
task1ID, _ := c.Add("@every 1m", func() error {
	fmt.Println("任務 1 執行")
	return nil
}, "任務 1")

// 任務 2：依賴任務 1，失敗時跳過繼續
task2ID, _ := c.Add("@every 1m", func() error {
	fmt.Println("任務 2 執行")
	return nil
}, "任務 2", []goCron.Wait{
	{ID: task1ID, State: goCron.Skip},
})

// 任務 3：依賴任務 2，失敗時停止，自定義超時 10 秒
_, _ = c.Add("@every 1m", func() error {
	fmt.Println("任務 3 執行")
	return nil
}, "任務 3", []goCron.Wait{
	{ID: task2ID, Delay: 10 * time.Second, State: goCron.Stop},
})
```

### 動態管理任務

```go
// 新增任務
id, _ := c.Add("@every 30s", func() {
	fmt.Println("可移除的任務")
}, "臨時任務")

// 移除單一任務
c.Remove(id)

// 移除所有任務
c.RemoveAll()

// 查詢目前任務列表
tasks := c.List()
for _, t := range tasks {
	fmt.Printf("任務 ID: %d, 描述: %s\n", t.ID, t.Description())
}
```

## API 參考

### New

```go
func New(c Config) (*cron, error)
```

建立新的排程器實例。`Config.Location` 為 nil 時使用系統本地時區。日誌優先輸出至 syslog，不可用時回退至 stderr。

### Config

```go
type Config struct {
	Location *time.Location
}
```

排程器設定。`Location` 指定排程計算使用的時區。

### Add

```go
func (c *cron) Add(spec string, action interface{}, arg ...interface{}) (int64, error)
```

新增排程任務。`spec` 接受標準 5 欄位 Cron 表達式或描述符。`action` 接受 `func()` 或 `func() error`。可選參數依序為：

| 參數類型 | 說明 |
|----------|------|
| `string` | 任務描述 |
| `time.Duration` | 任務執行超時時間 |
| `func()` | 超時時觸發的 callback |
| `[]Wait` | 依賴任務清單 |

回傳任務 ID 與錯誤。使用 `func()` 作為 action 時不支援依賴功能，需改用 `func() error`。

### Wait

```go
type Wait struct {
	ID    int64
	Delay time.Duration
	State WaitState
}
```

定義任務依賴關係。`ID` 為前置任務的 ID，`Delay` 為等待前置任務完成的超時時間（預設 1 分鐘），`State` 為前置任務失敗時的策略。

### WaitState

```go
const (
	Stop WaitState = iota  // 前置任務失敗時中止
	Skip                   // 前置任務失敗時跳過繼續
)
```

### Start

```go
func (c *cron) Start()
```

啟動排程器。重複呼叫為安全操作，不會建立額外的排程迴圈。

### Stop

```go
func (c *cron) Stop() context.Context
```

停止排程器並回傳 Context，當所有執行中的任務完成後 Context 會被取消。

### Remove

```go
func (c *cron) Remove(id int64)
```

移除指定 ID 的任務。

### RemoveAll

```go
func (c *cron) RemoveAll()
```

移除所有任務。

### List

```go
func (c *cron) List() []task
```

回傳所有已啟用的任務列表。

### 排程語法

| 格式 | 範例 | 說明 |
|------|------|------|
| 5 欄位 Cron | `30 2 * * *` | 分 時 日 月 週 |
| 步進 | `*/15 * * * *` | 每 15 分鐘 |
| 範圍 | `0 9 * * 1-5` | 週一至週五上午 9 點 |
| 列表 | `0 0 * * 1,3,5` | 週一、三、五午夜 |
| `@yearly` | - | 每年 1 月 1 日 00:00 |
| `@monthly` | - | 每月 1 日 00:00 |
| `@weekly` | - | 每週日 00:00 |
| `@daily` | - | 每天 00:00 |
| `@hourly` | - | 每小時 00 分 |
| `@every` | `@every 30s` | 固定間隔（最小 30 秒） |

***

©️ 2025 [邱敬幃 Pardn Chiu](https://linkedin.com/in/pardnchiu)
