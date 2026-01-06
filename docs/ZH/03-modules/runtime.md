# 模块设计: Runtime

> 系统的心脏 - 主循环、状态管理、异常处理

---

## 1. 概述

Runtime 是 gm-agent 的**核心控制面**，负责：
- 主循环执行
- 状态转换 (Reducer)
- 命令分发 (Dispatcher)
- 异常处理与恢复

> ⚠️ **本模块是整个系统的热路径，任何 bug 都可能导致系统死循环或崩溃。**

---

## 2. 主循环

### 2.1 伪代码（规范性）

> **数据类型定义**: 见 [data-model.md](../02-architecture/data-model.md)

```go
func (r *Runtime) Run(ctx context.Context) error {
    // 1. 恢复状态 (如果有 checkpoint)
    if err := r.recover(ctx); err != nil {
        return fmt.Errorf("recovery failed: %w", err)
    }
    
    // 2. 主循环
    for step := 0; step < r.config.MaxSteps; step++ {
        // 2.1 检查取消
        if ctx.Err() != nil {
            return r.gracefulShutdown(ctx)
        }
        
        // 2.2 处理 pending commands (来自 Reducer 产出)
        if len(r.pendingCommands) > 0 {
            cmdsToExecute := r.pendingCommands
            r.pendingCommands = nil  // 清空
            
            events, err := r.dispatch(ctx, cmdsToExecute)
            if err != nil {
                r.recordError(err)
            }
            
            // 应用事件
            for _, event := range events {
                if err := r.applyEvent(ctx, event); err != nil {
                    return err
                }
            }
            continue  // 继续处理可能产生的新 pending commands
        }
        
        // 2.3 选择目标 (Attention)
        goal, err := r.selectGoal()
        if err != nil {
            return r.handleFatalError(err)
        }
        if goal == nil {
            return nil  // 所有目标完成
        }
        
        // 2.4 请求 LLM 决策 (Decide)
        decision, err := r.decide(ctx, goal)
        if err != nil {
            if r.shouldRetry(err) {
                continue
            }
            return r.handleFatalError(err)
        }
        
        // 2.5 执行 LLM 决策命令 (Act)
        events, err := r.dispatch(ctx, decision.Commands)
        if err != nil {
            r.recordError(err)
            // 不终止，继续下一步让 LLM 决定如何处理
        }
        
        // 2.6 应用事件更新状态 (Observe)
        for _, event := range events {
            if err := r.applyEvent(ctx, event); err != nil {
                return err
            }
        }
        
        // 2.7 检查是否需要 checkpoint (Commit)
        if r.shouldCheckpoint(step) {
            if err := r.checkpoint(ctx); err != nil {
                r.log.Warn("checkpoint failed", "error", err)
                // 不终止，checkpoint 失败不是致命错误
            }
        }
    }
    
    return &MaxStepsExceededError{Steps: r.config.MaxSteps}
}

// applyEvent 应用单个事件到状态
func (r *Runtime) applyEvent(ctx context.Context, event Event) error {
    newState, cmds, err := r.safeReduce(ctx, r.state, event)
    if err != nil {
        return r.handleReducerError(err, event)
    }
    r.state = newState
    
    // Reducer 产出的命令加入 pending 队列
    // 这些命令会在下一轮主循环开始时执行
    r.pendingCommands = append(r.pendingCommands, cmds...)
    
    return nil
}
```

### 2.2 终止条件

| 条件 | 行为 | 返回值 |
| :--- | :--- | :--- |
| `ctx.Done()` | 优雅关闭 | `context.Canceled` 或 `context.DeadlineExceeded` |
| 所有目标完成 | 正常退出 | `nil` |
| 达到 MaxSteps | 超限退出 | `MaxStepsExceededError` |
| Reducer panic | 致命错误 | `ReducerPanicError` |
| 恢复失败 | 无法启动 | 原始错误 |

### 2.3 配置

```go
type RuntimeConfig struct {
    MaxSteps           int           `yaml:"max_steps"`            // 最大步数，默认 100
    CheckpointInterval int           `yaml:"checkpoint_interval"`  // 每 N 步 checkpoint，默认 10
    CheckpointTimeout  time.Duration `yaml:"checkpoint_timeout"`   // checkpoint 超时，默认 5s
    DecisionTimeout    time.Duration `yaml:"decision_timeout"`     // LLM 决策超时，默认 60s
    DispatchTimeout    time.Duration `yaml:"dispatch_timeout"`     // 命令执行超时，默认 300s
}

var DefaultRuntimeConfig = RuntimeConfig{
    MaxSteps:           100,
    CheckpointInterval: 10,
    CheckpointTimeout:  5 * time.Second,
    DecisionTimeout:    60 * time.Second,
    DispatchTimeout:    300 * time.Second,
}
```

---

## 3. 异常处理

### 3.1 错误分类

```go
// ErrorSeverity 定义见 data-model.md
func ClassifyError(err error) ErrorSeverity {
    switch {
    case errors.Is(err, context.DeadlineExceeded):
        return SeverityRetryable
    case errors.Is(err, ErrRateLimit):
        return SeverityRetryable
    case errors.Is(err, ErrToolFailed):
        return SeverityRecoverable  // 让 LLM 决定下一步
    case errors.Is(err, ErrReducerPanic):
        return SeverityFatal
    case errors.Is(err, ErrStateCorrupted):
        return SeverityFatal
    default:
        return SeverityRecoverable
    }
}
```

### 3.2 Reducer Panic 处理

```go
func (r *Runtime) safeReduce(ctx context.Context, state *State, event Event) (newState *State, cmds []Command, err error) {
    defer func() {
        if p := recover(); p != nil {
            err = &ReducerPanicError{
                Panic:      p,
                Event:      event,
                StackTrace: string(debug.Stack()),
            }
            // 记录到审计日志
            r.audit.Log(ctx, AuditEvent{
                Type:   "reducer.panic",
                Action: event.EventType(),
                Result: "error",
                Metadata: map[string]any{
                    "panic":       fmt.Sprintf("%v", p),
                    "stack_trace": err.(*ReducerPanicError).StackTrace,
                },
            })
        }
    }()
    
    return r.reducer(state, event)
}
```

### 3.3 命令执行失败处理

```go
func (r *Runtime) dispatch(ctx context.Context, cmds []Command) ([]Event, error) {
    var allEvents []Event
    var lastErr error
    
    for _, cmd := range cmds {
        events, err := r.dispatcher.Execute(ctx, cmd)
        if err != nil {
            // 记录错误但不终止
            lastErr = err
            
            // 生成错误事件，让 LLM 知道发生了什么
            errorEvent := &ErrorEvent{
                BaseEvent: NewBaseEvent("error", "system", ""),
                CommandID: cmd.CommandID(),
                Error:     err.Error(),
                Severity:  ClassifyError(err),
            }
            allEvents = append(allEvents, errorEvent)
            
            // 如果是致命错误，立即停止
            if ClassifyError(err) == SeverityFatal {
                return allEvents, err
            }
            
            continue
        }
        
        allEvents = append(allEvents, events...)
    }
    
    return allEvents, lastErr
}
```

### 3.4 异常分支处理表

| 场景 | 处理方式 | 状态影响 |
| :--- | :--- | :--- |
| **LLM 超时** | 重试 3 次，然后生成 `LLMTimeoutEvent` | 不变 |
| **LLM 返回空** | 生成 `EmptyResponseEvent` | 不变 |
| **LLM 返回非法 JSON** | 尝试修复，失败则生成 `ParseErrorEvent` | 不变 |
| **工具执行失败** | 生成 `ToolFailedEvent`，让 LLM 决定 | 不变 |
| **工具超时** | 终止工具，生成 `ToolTimeoutEvent` | 不变 |
| **Patch 冲突** | 生成 `PatchConflictEvent`，让 LLM 决定 | 不变 |
| **Reducer Panic** | 记录审计日志，**立即终止** | 保持崩溃前状态 |
| **用户取消** | 优雅关闭，保存 checkpoint | checkpoint 状态 |

---

## 4. 状态回滚策略

### 4.1 设计原则

> **gm-agent 采用"事件不可删除"原则：一旦事件写入，永不回滚。**

错误通过追加"补偿事件"来修复，而不是删除已有事件。

### 4.2 补偿事件示例

```go
// Patch 失败后，生成回滚事件
type PatchRollbackEvent struct {
    ID          string `json:"id"`
    OriginalID  string `json:"original_id"`  // 失败的 Patch 事件 ID
    Reason      string `json:"reason"`
}

// 状态恢复到 Patch 之前
func (r *Reducer) handlePatchRollback(state *State, event *PatchRollbackEvent) (*State, []Command, error) {
    // 从备份恢复文件
    cmd := &RestoreBackupCommand{
        PatchID: event.OriginalID,
    }
    return state, []Command{cmd}, nil
}
```

### 4.3 Checkpoint 恢复

```go
func (r *Runtime) recover(ctx context.Context) error {
    // 1. 加载最新 checkpoint
    cp, err := r.store.LoadLatestCheckpoint(ctx)
    if err != nil {
        if errors.Is(err, ErrNoCheckpoint) {
            // 首次运行，初始化空状态
            r.state = NewState()
            return nil
        }
        return err
    }
    
    // 2. 恢复状态
    state, err := r.store.LoadState(ctx, cp.StateVersion)
    if err != nil {
        return fmt.Errorf("load state failed: %w", err)
    }
    r.state = state
    
    // 3. 重放 checkpoint 之后的事件
    events, err := r.store.GetEventsSince(ctx, cp.LastEventID)
    if err != nil {
        return fmt.Errorf("load events failed: %w", err)
    }
    
    r.log.Info("replaying events", "count", len(events))
    for _, event := range events {
        newState, _, err := r.safeReduce(ctx, r.state, event)
        if err != nil {
            // 重放失败是致命错误
            return fmt.Errorf("replay failed at event %s: %w", event.EventID(), err)
        }
        r.state = newState
    }
    
    return nil
}
```

---

## 5. 资源释放

### 5.1 关闭顺序

```go
func (r *Runtime) gracefulShutdown(ctx context.Context) error {
    r.log.Info("graceful shutdown started")
    
    // 1. 停止接收新事件
    r.inbox.Close()
    
    // 2. 等待当前命令执行完成 (带超时)
    shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()
    
    r.dispatcher.WaitAll(shutdownCtx)
    
    // 3. 保存最终 checkpoint
    if err := r.checkpoint(shutdownCtx); err != nil {
        r.log.Error("final checkpoint failed", "error", err)
    }
    
    // 4. 关闭 LLM 连接
    r.llm.Close()
    
    // 5. 关闭存储
    r.store.Close()
    
    // 6. 等待所有 goroutine 退出
    r.wg.Wait()
    
    r.log.Info("graceful shutdown completed")
    return nil
}
```

### 5.2 Goroutine 管理

```go
type Runtime struct {
    wg     sync.WaitGroup
    cancel context.CancelFunc
    
    // 所有后台 goroutine 必须通过这个方法启动
    go func(r *Runtime) spawn(fn func()) {
        r.wg.Add(1)
        go func() {
            defer r.wg.Done()
            fn()
        }()
    }
}
```

---

## 6. 可观测性

### 6.1 指标

```go
var (
    runtimeStepsTotal = prometheus.NewCounter(prometheus.CounterOpts{
        Name: "gm_runtime_steps_total",
        Help: "Total number of runtime steps executed",
    })
    
    runtimeStepDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
        Name:    "gm_runtime_step_duration_seconds",
        Help:    "Duration of each runtime step",
        Buckets: prometheus.DefBuckets,
    })
    
    runtimeErrorsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
        Name: "gm_runtime_errors_total",
        Help: "Total number of runtime errors by severity",
    }, []string{"severity"})
    
    runtimeCheckpointsTotal = prometheus.NewCounter(prometheus.CounterOpts{
        Name: "gm_runtime_checkpoints_total",
        Help: "Total number of checkpoints saved",
    })
)
```

### 6.2 结构化日志

```go
// 每个主要阶段都有日志
r.log.Info("step started", 
    "step", step,
    "goal_id", goal.ID,
    "goal_type", goal.Type,
)

r.log.Debug("decision received",
    "step", step,
    "commands", len(decision.Commands),
)

r.log.Info("step completed",
    "step", step,
    "events", len(events),
    "duration_ms", time.Since(stepStart).Milliseconds(),
)
```

---

## 7. 测试边界

### 7.1 单元测试要求

| 测试场景 | 验收标准 |
| :--- | :--- |
| Reducer 纯函数 | 相同输入产生相同输出，无副作用 |
| Reducer panic | 正确捕获并返回错误 |
| MaxSteps 限制 | 达到限制后返回错误 |
| Context 取消 | 收到取消信号后优雅退出 |
| Checkpoint 恢复 | 从 checkpoint 恢复后状态一致 |
| 事件重放 | 重放后状态与原运行一致 |

### 7.2 测试覆盖率

```
runtime/
├── runtime.go          # 主循环，覆盖率 >= 90%
├── reducer.go          # Reducer，覆盖率 >= 95%
├── dispatcher.go       # 分发器，覆盖率 >= 85%
├── checkpoint.go       # Checkpoint，覆盖率 >= 90%
└── recovery.go         # 恢复逻辑，覆盖率 >= 90%
```

---

## 8. 已知限制

> ⚠️ **必须阅读**

1. **MaxSteps 是硬限制**：达到后任务会被强制终止，无论是否完成。调用方有责任设置合理的值。

2. **Checkpoint 不保证原子性（FS Store）**：使用 FS Store 时，如果在写 checkpoint 中途崩溃，可能导致状态不一致。**生产环境必须使用 SQLite 或 PostgreSQL**。

3. **Reducer 必须是纯函数**：如果 Reducer 有副作用（如写文件），系统行为不可预测。所有副作用必须通过 Command 执行。

4. **不支持分布式**：当前设计假设单进程运行。多实例需要外部协调（如 Redis/etcd 分布式锁）。
