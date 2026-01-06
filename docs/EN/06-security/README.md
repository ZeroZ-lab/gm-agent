# Security Overview

> Key practices for safe and auditable execution

## Secrets & Keys
- Keep provider keys in environment variables or config with strict file permissions.
- Redact keys and sensitive payloads in logs and persisted events.

## Access Control
- Policy gate controls tool usage (allow/deny/ask) per actor/subject.
- Role-based access for CLI/API keys to limit dangerous operations.

## Audit
- All events and tool calls are persisted with timestamps and request IDs.
- Artifacts and patches include authorship metadata for traceability.

## Safety Guards
- Command-injection protections for shell calls; reject suspicious arguments.
- Binary detection before generating diffs; refuse unsafe file types.
- Lock timeouts to avoid deadlocks; checkpoints to enable recovery after crashes.
