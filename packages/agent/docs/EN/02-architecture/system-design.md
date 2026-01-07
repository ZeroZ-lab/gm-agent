# System Design

> Core design principles and runtime mechanics

---

## 1. Design Philosophy

### 1.1 Event-driven durable runtime
- Everything is expressed as events; reducers turn `(state, event)` into `(state, commands)`.
- Durability first: every event is appended and can be replayed after crashes.
- Deterministic core keeps the LLM decision-making but not control flow.

### 1.2 Minimal autonomous loop
1. Receive normalized event.
2. Persist event.
3. Reducer produces next state + commands.
4. Dispatcher executes commands (LLM/tools/patch).
5. Write outputs and checkpoints at safe points.

---

## 2. Reducer Pattern

### 2.1 Interface
- Pure function: `func Reduce(state, event) (newState, commands, err)`
- No side effects; all IO happens in commands.

### 2.2 Events
- Core events include: `UserMessage`, `ToolResult`, `LLMResponse`, `Checkpointed`, `Canceled`, `Preempted`.
- Events carry metadata (actor, subject, priority, event_id) for replay and auditing.

### 2.3 Commands
- Command categories: `CallLLM`, `CallTool`, `ApplyPatch`, `LogArtifact`, `Checkpoint`.
- Commands include timeout/budget and target subjects to allow scheduling.

---

## 3. Dispatcher

### 3.1 Responsibilities
- Route commands to the correct executor (LLM, tool runner, patch engine).
- Enforce policies (timeouts, retries, circuit breakers).
- Emit events for every command outcome for downstream reducers.

### 3.2 Implementation Notes
- Use typed executors per command type.
- Structured logging and redaction around every external call.
- Map command correlation IDs back to originating events for tracing.

---

## 4. Scheduler

### 4.1 Concurrency Model
- Worker pool with configurable concurrency.
- Single-writer locks per subject; reads can run in parallel.
- Priority-aware queue for different actors/subjects.

### 4.2 Locks
- Subject-level mutex with timeout to prevent deadlocks.
- Preemption occurs only at safe points to keep atomicity.

### 4.3 Task Priority
- Priority derived from event metadata; high-priority items jump the queue.

---

## 5. Checkpoint & Recovery

### 5.1 Safe Points
- Checkpoints are created after critical milestones (tool results, patch apply).
- Atomically persist snapshot + last event offset.

### 5.2 Recovery Flow
1. Load latest checkpoint.
2. Replay events after checkpoint to rebuild state.
3. Resume scheduler with reconstructed state.

---

## 6. Multi-actor Handling

### 6.1 Input Normalization
- Different triggers (user, file watcher, webhook) are normalized into events.
- Metadata distinguishes actors/subjects to support conflict resolution.

### 6.2 Conflict Handling
- Locking ensures serialized writes to the same subject.
- Preempt and cancel events allow cooperative interruption.

---

## 7. Go Best Practices

### 7.1 Context Propagation
- Context is mandatory for every executor; cancel signals propagate to goroutines.

### 7.2 Goroutine Management
- Use worker pools; avoid unbounded goroutines.
- Shutdown hooks drain queues gracefully.

### 7.3 Error Handling
- Wrap errors with context; categorize retryable vs fatal.
- Never panic on external input; prefer controlled fallback.

---

## 8. Engineering Principles
- Deterministic core; side effects isolated.
- Observability by design (structured logs, metrics, traces later).
- Security by default: permissions, redaction, auditability.

---

## Next Steps
- [Data Model](./data-model.md)
- [Module Guides](../03-modules/runtime.md)
