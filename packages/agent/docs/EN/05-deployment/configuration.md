# Configuration

> Key configuration fields and their meaning

## LLM Providers
- `providers.openai.api_key`, `providers.openai.base_url`, `providers.openai.model`
- `providers.anthropic.api_key`, `providers.anthropic.base_url`, `providers.anthropic.model`
- Retry/backoff and timeout settings per provider.

## Policies
- Default policy for tools (allow/deny/ask) and per-actor overrides.
- Shell allowlist/denylist, file path restrictions, and max runtime per command.

## Runtime
- Concurrency limits, worker pool size, lock timeout, checkpoint interval.
- Workspace paths for events, snapshots, and artifacts.

## Logging
- Log level, JSON output toggle, redaction settings, request ID enablement.
