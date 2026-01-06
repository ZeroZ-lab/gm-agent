# CLI Commands

> Command-line entrypoints for gm-agent

## gm run
- Usage: `gm run "<prompt>"`
- Sends a user message to the runtime and streams the final response.
- Flags: `--config` for custom config path, `--workspace` to set working dir.

## gm status
- Show current or most recent task status, including active locks and checkpoints.

## gm history
- List historical sessions with timestamps and outcomes.

## Notes
- CLI uses the same event pipeline as HTTP/MCP; outputs are fully persisted.
