# LLM Module

> Unified gateway and provider adapters

## Gateway
- Normalizes prompts, tool definitions, and stop conditions across providers.
- Streams responses (SSE) with support for mid-stream cancellation.
- Adds retry with exponential backoff for retryable errors; respects `Retry-After` on 429.

## Providers
- OpenAI: gpt-4 / gpt-3.5-turbo with tool calls.
- Anthropic: claude-sonnet-4-20250514 with `tool_use` semantics.
- Extensible interface for adding new providers (API key handling, headers, JSON mapping).

## Token Budget
- Tracks token usage per session and enforces budgets from goals/tasks.
- Provides automatic context compaction when exceeding limits.

## Observability
- Structured logs for every request/response chunk with redacted secrets.
- Optional metrics: latency, token count, cost estimation.
