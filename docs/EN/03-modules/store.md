# Store Module

> Persistence for events, snapshots, and artifacts

## Interfaces
- `AppendEvent` with idempotent `event_id`.
- `ListEvents`/`IterEvents` for replay.
- `SaveCheckpoint` and `LoadLatestCheckpoint`.
- Artifact operations: save/read/list/delete with metadata.

## Implementations
- **Filesystem (MVP)**: JSONL event log + snapshot files, atomic write-rename to avoid partial files.
- **Databases (future)**: SQLite/PostgreSQL adapters with transactions and outbox pattern.

## Reliability
- Atomic writes to prevent corruption.
- Background checkpointing every N events or T seconds.
- Outbox pattern to store events and side effects together.
