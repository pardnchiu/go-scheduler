# go-scheduler - Architecture

> Back to [README](../README.md)

## Overview

```mermaid
graph TB
    App[Application] --> Cron[Cron Scheduler]
    Cron --> Parser[Expression Parser]
    Cron --> Heap[Task Min-Heap]
    Cron --> Depend[Dependency Subsystem]
    Depend --> Manager[Depend Manager]
    Depend --> Workers[Worker Pool]
    Heap --> Runner[Task Runner]
    Workers --> Runner
```

## Module: Cron Scheduler

Owns lifecycle, task registration, and the time-driven main loop.

```mermaid
graph TB
    subgraph Cron
        New[New Config] --> Instance[cron instance]
        Instance --> Start[Start]
        Instance --> Stop[Stop]
        Instance --> Add[Add]
        Instance --> Remove[Remove / RemoveAll]
        Instance --> List[List]
        Start --> Loop[Main event loop]
        Loop --> Timer[Nearest-task timer]
        Loop --> Channels[add / remove / stop channels]
    end
    App[Caller] --> New
    App --> Start
    App --> Add
```

## Module: Expression Parser

Turns string specs into `schedule` implementations.

```mermaid
graph TB
    subgraph Parser
        Parse[parse spec] --> Descriptor{Starts with @?}
        Descriptor -->|yes| Desc[parseDescriptor]
        Descriptor -->|no| Cron5[parseCron 5 fields]
        Desc --> Delay[delayScheduleResult]
        Desc --> Fixed[scheduleResult fixed time]
        Cron5 --> Field[parseField]
        Field --> All[All *]
        Field --> Step[Step */n]
        Field --> Range[Range n-m]
        Field --> List[List a,b,c]
        Field --> Value[Single n]
    end
    Add[Add] --> Parse
    Delay --> Next1[next = now + delay]
    Fixed --> Next2[next = next matching minute]
```

## Module: Task Min-Heap

Orders tasks by `next` so the main loop always handles the nearest due task.

```mermaid
graph TB
    subgraph Heap
        H[taskHeap] --> Less[Less: earlier next first]
        H --> Push[Push new task]
        H --> Pop[Pop due task]
        H --> Remove[Remove by index]
    end
    Loop[Main loop] --> Pop
    Add[Add task] --> Push
    RemoveAPI[Remove API] --> Remove
```

## Module: Dependency Subsystem

When a task has `after` dependencies, the worker pool waits for prerequisites before running it.

```mermaid
graph TB
    subgraph Depend
        D[depend] --> Queue[Wait queue]
        D --> W1[Worker 1]
        D --> W2[Worker 2]
        D --> Wn[Worker N = NumCPU]
        Queue --> W1
        Queue --> W2
        Queue --> Wn
        W1 --> RunAfter[runAfter]
        RunAfter --> Manager[dependManager]
        Manager --> Check[check dependency state]
        Manager --> Wait[wait with timeout]
        Manager --> Update[update result]
    end
    CronRun[cron.run] -->|has deps| Queue
    CronRun -->|no deps| Direct[runAfter direct]
```

## Data Flow

```mermaid
sequenceDiagram
    participant App as Application
    participant Cron as Cron
    participant Heap as Task Heap
    participant Depend as Dependency Subsystem
    participant Task as Task action

    App->>Cron: New / Add / Start
    Cron->>Heap: compute next and Init
    loop Main loop
        Cron->>Heap: wait for nearest next
        Heap-->>Cron: due task
        alt no dependencies
            Cron->>Task: run directly
        else has dependencies
            Cron->>Depend: addWait
            Depend->>Depend: wait for prerequisites
            Depend->>Task: run after prerequisites complete
        end
        Task-->>Cron: success / failure / timeout
        Cron->>Heap: compute next next and Push
    end
    App->>Cron: Stop
    Cron-->>App: context cancels after in-flight finish
```

## State Machine

```mermaid
stateDiagram-v2
    [*] --> TaskPending: Add
    TaskPending --> TaskRunning: trigger run
    TaskRunning --> TaskCompleted: action success
    TaskRunning --> TaskFailed: error / panic / timeout / dependency failure
    TaskCompleted --> TaskPending: reschedule recurring task
    TaskFailed --> TaskPending: reschedule recurring task
    TaskPending --> [*]: Remove / disable
```

***

©️ 2025 [邱敬幃 Pardn Chiu](https://linkedin.com/in/pardnchiu)
