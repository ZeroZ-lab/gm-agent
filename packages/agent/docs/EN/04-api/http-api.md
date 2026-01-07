# HTTP API

> REST surface for integrating gm-agent

## Endpoints
- `POST /api/v1/sessions`: start a session with prompt and options (actors, subjects, budgets).
- `GET /api/v1/sessions/{id}`: fetch session status and current state summary.
- `GET /api/v1/sessions/{id}/events`: stream events or fetch paginated history.
- `POST /api/v1/sessions/{id}/cancel`: request cancellation/preemption.

### OpenAPI
- `GET /api/openapi.json`: machine-readable contract (OpenAPI 3.0). Mirrors OpenCode-style envelope.

### Health
- `GET /healthz`: simple readiness indicator for load balancers.

## Authentication & Security
- API key header required; keys map to policy profiles.
- Rate limits enforced per key; request IDs returned for tracing.
- Sensitive fields (API keys, secrets) are redacted in logs.

## Responses
- JSON with `request_id`, `data`, and `error` envelope.
- SSE streaming supported for long-running sessions.
