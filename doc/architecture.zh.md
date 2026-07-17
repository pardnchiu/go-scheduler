# go-scheduler - 架構

> 返回 [README](./README.zh.md)

## 概覽

```mermaid
graph TB
    App[應用程式] --> Cron[Cron 排程器]
    Cron --> Parser[表達式解析器]
    Cron --> Heap[任務最小堆]
    Cron --> Depend[依賴子系統]
    Depend --> Manager[依賴管理器]
    Depend --> Workers[Worker 池]
    Heap --> Runner[任務執行器]
    Workers --> Runner
```

## 模組：Cron 排程器

負責生命週期、任務登錄與時間觸發的主迴圈。

```mermaid
graph TB
    subgraph Cron
        New[New Config] --> Instance[cron 實例]
        Instance --> Start[Start]
        Instance --> Stop[Stop]
        Instance --> Add[Add]
        Instance --> Remove[Remove / RemoveAll]
        Instance --> List[List]
        Start --> Loop[主事件迴圈]
        Loop --> Timer[最近任務計時器]
        Loop --> Channels[add / remove / stop 通道]
    end
    App[呼叫端] --> New
    App --> Start
    App --> Add
```

## 模組：表達式解析器

將字串規格轉成 `schedule` 介面實作。

```mermaid
graph TB
    subgraph Parser
        Parse[parse spec] --> Descriptor{以 @ 開頭?}
        Descriptor -->|是| Desc[parseDescriptor]
        Descriptor -->|否| Cron5[parseCron 五欄位]
        Desc --> Delay[delayScheduleResult]
        Desc --> Fixed[scheduleResult 固定時間]
        Cron5 --> Field[parseField]
        Field --> All[All *]
        Field --> Step[Step */n]
        Field --> Range[Range n-m]
        Field --> List[List a,b,c]
        Field --> Value[單值 n]
    end
    Add[Add] --> Parse
    Delay --> Next1[next = now + delay]
    Fixed --> Next2[next = 下一符合分鐘]
```

## 模組：任務最小堆

以 `next` 時間排序，確保主迴圈永遠處理最近到期的任務。

```mermaid
graph TB
    subgraph Heap
        H[taskHeap] --> Less[Less: next 較早者優先]
        H --> Push[Push 新任務]
        H --> Pop[Pop 到期任務]
        H --> Remove[Remove 指定索引]
    end
    Loop[主迴圈] --> Pop
    Add[新增任務] --> Push
    RemoveAPI[Remove API] --> Remove
```

## 模組：依賴子系統

當任務具有 `after` 依賴時，由 worker 池等待前置完成再執行。

```mermaid
graph TB
    subgraph Depend
        D[depend] --> Queue[Wait 佇列]
        D --> W1[Worker 1]
        D --> W2[Worker 2]
        D --> Wn[Worker N = NumCPU]
        Queue --> W1
        Queue --> W2
        Queue --> Wn
        W1 --> RunAfter[runAfter]
        RunAfter --> Manager[dependManager]
        Manager --> Check[check 依賴狀態]
        Manager --> Wait[wait 含逾時]
        Manager --> Update[update 結果]
    end
    CronRun[cron.run] -->|有依賴| Queue
    CronRun -->|無依賴| Direct[runAfter 直接執行]
```

## 資料流

```mermaid
sequenceDiagram
    participant App as 應用程式
    participant Cron as Cron
    participant Heap as 任務堆
    participant Depend as 依賴子系統
    participant Task as 任務 action

    App->>Cron: New / Add / Start
    Cron->>Heap: 計算 next 並 Init
    loop 主迴圈
        Cron->>Heap: 等待最近 next
        Heap-->>Cron: 到期任務
        alt 無依賴
            Cron->>Task: 直接執行
        else 有依賴
            Cron->>Depend: addWait
            Depend->>Depend: wait 前置任務
            Depend->>Task: 前置完成後執行
        end
        Task-->>Cron: 成功 / 失敗 / 逾時
        Cron->>Heap: 計算下次 next 並 Push
    end
    App->>Cron: Stop
    Cron-->>App: context 於 in-flight 結束後取消
```

## 狀態機

```mermaid
stateDiagram-v2
    [*] --> TaskPending: Add
    TaskPending --> TaskRunning: 觸發執行
    TaskRunning --> TaskCompleted: action 成功
    TaskRunning --> TaskFailed: error / panic / 逾時 / 依賴失敗
    TaskCompleted --> TaskPending: 週期任務重排
    TaskFailed --> TaskPending: 週期任務重排
    TaskPending --> [*]: Remove / 停用
```

***

©️ 2025 [邱敬幃 Pardn Chiu](https://linkedin.com/in/pardnchiu)
