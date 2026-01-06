# Data Model

> Key runtime entities, events, and storage schemas

---

## 1. Overview
- Strongly typed Go structs keep the runtime deterministic.
- Event sourcing: immutable event log + periodic snapshots.

## 2. State Model

### 2.1 State (system state)
- Fields: `session_id`, `actors`, `subjects`, `goals`, `tasks`, `context_window`, `locks`, `last_event_id`, `created_at`, `updated_at`.
- Tracks active tasks, resources in use, and accumulated artifacts.

### 2.2 Goal
- Represents user intent and acceptance criteria.
- Fields include `id`, `title`, `description`, `priority`, `status`, `metrics` (quality, latency, cost budgets).

### 2.3 Task
- Executable unit created by reducer.
- Attributes: `id`, `goal_id`, `subject`, `assignee` (Main/Sub agent), `status`, `steps`, `retries`, `deadline`, `logs`.

### 2.4 ContextWindow
- Rolling window of messages and tool results with token budget.
- Holds `messages`, `traces`, and truncation strategy metadata.

### 2.5 Lock
- Subject-level lock info: `subject`, `owner`, `mode` (read/write), `expires_at`, `waiting_queue`.

## 3. Event Model

### 3.1 Event (base)
- Fields: `event_id`, `parent_id`, `type`, `actor`, `subject`, `priority`, `payload`, `created_at`.
- Idempotent by `event_id` to support replay.

### 3.2 Event Types
- User/trigger events: `UserMessage`, `FileChanged`, `TimerTriggered`, `WebhookEvent`.
- Runtime events: `LLMRequested`, `LLMResponded`, `ToolCalled`, `ToolReturned`, `PatchApplied`, `Checkpointed`.
- Control events: `Preempted`, `Canceled`, `Forked`.

## 4. Command Model

### 4.1 Command Interface
- `type`, `command_id`, `subject`, `timeout`, `retries`, `budget`, `payload`.

### 4.2 Command Types
- `CallLLM`: prompt, tools, stop sequences, budget.
- `CallTool`: tool name, arguments, policy context.
- `ApplyPatch`: diff, dry-run flag, rollback token.
- `LogArtifact`: path, media type, content hash/URI.
- `Checkpoint`: snapshot hint.

## 5. Storage Model

### 5.1 Checkpoint
- Snapshot of state with `version`, `last_event_id`, `state_blob`, `created_at`.

### 5.2 Artifact
- Metadata: `artifact_id`, `subject`, `path`, `media_type`, `size`, `checksum`, `created_by`, `created_at`.

## 6. Store Interface
- `AppendEvent(event)` with idempotency.
- `ListEvents(after_id, limit)` for replay.
- `SaveCheckpoint(checkpoint)` atomically with log when possible.
- `LoadLatestCheckpoint()` and `LoadArtifacts(subject)`.

## 7. Tool Model

### 7.1 Tool Definition
- `name`, `description`, `input_schema`, `output_schema`, `policy` (allow/deny/ask), `timeout`, `examples`.
- Executor contract returns structured result + logs.

## 8. Auxiliary Types
- `Actor` (id, role, permissions), `Subject` (resource identifier), `Trace` (structured log entry), `Budget` (tokens, time, cost).

## 9. Relationships
- Event log drives reducer → new state → commands → new events.
- Checkpoints bind log offsets to state for fast recovery.
- Tasks belong to goals; locks guard subjects; artifacts attach to subjects/goals.
