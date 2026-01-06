# Runtime Module

> State machine, dispatcher, scheduler, and recovery loop

## Responsibilities
- Owns the main reducer function to convert events into new state + commands.
- Manages worker pool, subject locks, and priority queues.
- Ensures every command execution emits a new event for replayability.
- Creates checkpoints and performs crash recovery from snapshots + logs.

## Flow
1. Receive normalized event from CLI/API/trigger.
2. Append to event log (idempotent by `event_id`).
3. Reduce to new state and commands.
4. Enqueue commands; dispatcher executes with policy enforcement.
5. On safe points, persist checkpoints and artifacts.

## Safety & Reliability
- Subject-level locks serialize writes while allowing concurrent reads.
- Timeouts on locks and commands to avoid deadlocks.
- Replay mechanism rebuilds state on restart using last checkpoint + tail events.
- Structured logging with request IDs and redaction by default.

## Extensibility
- Reducer and scheduler expose interfaces for custom policies.
- Commands are typed to allow new executors (e.g., WASM, MCP).
