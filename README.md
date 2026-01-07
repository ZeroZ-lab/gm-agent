# gm-agent

Enterprise-grade, extensible agent runtime and server framework with a durable event-driven core, pluggable tool system, and LLM gateway exposed via a unified HTTP API.

Read this document in Chinese: `README.zh-CN.md`

## Capabilities

- Durable runtime (Reducer + Dispatcher + Checkpoint)
- LLM gateway with multi-provider adapters
- Tool registry, policy gating, and isolated execution
- Session management and event storage (FS Store)
- Standardized HTTP API (Swagger included)

## Architecture

Detailed architecture and design:
- `packages/agent/docs/EN/02-architecture/README.md`
- `packages/agent/docs/EN/02-architecture/system-design.md`

## Quick Start

Requires Go 1.25.5.

```bash
make build
make run
```

HTTP API listens on `:8080` by default.

## Configuration

- Example config: `packages/agent/config.yaml.example`
- Priority: environment variables (`GM_` prefix) > config file > defaults
- Configurable items: HTTP address, API key, security policy, LLM provider, etc.

## API Documentation

- OpenAPI: `packages/agent/docs/swagger.yaml`
- Swagger JSON: `packages/agent/docs/swagger.json`

## Monorepo Layout

```
packages/
  agent/      # Core service and runtime
  web/        # Web client
  tui/        # TUI client
  desktop/    # Desktop client
```

## Build and Test

```bash
make test
make lint
make verify
```

## Security and Access Control

Security policies support tool access control, filesystem and network restrictions, and optional API key protection. See the configuration and security module documentation.
