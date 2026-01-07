# Feature Requirements

> Phased requirement list with acceptance criteria

---

## Phase 1: MVP

**Goal**: Run the full pipeline with a single Agent
**Timeline**: 2-4 weeks

### 1.1 Runtime Core

| Requirement | Priority | Acceptance | Dependency |
| :--- | :---: | :--- | :--- |
| Reducer pattern | P0 | `(state, event) -> (state, commands)` pure-function tests pass | None |
| Dispatcher | P0 | Can dispatch to LLM/Tool/Patch executors | Reducer |
| Worker pool | P0 | Configurable max concurrency, no goroutine leaks | None |
| Subject lock | P0 | Writes to the same resource are serialized, reads parallelized | None |
| **Deadlock prevention** | P0 | Lock timeout: fail after 30s if lock not acquired | Subject lock |

### 1.2 Storage

| Requirement | Priority | Acceptance | Dependency |
| :--- | :---: | :--- | :--- |
| FS store | P0 | Append to events.jsonl; overwrite state.json | None |
| Artifact management | P0 | Save/read/list/delete supported | FS store |
| Idempotent `event_id` | P0 | Duplicate `event_id` is not written twice | FS store |
| **Atomic writes** | P0 | Write failures leave no partial files (write-rename) | FS store |

### 1.3 Tooling

| Requirement | Priority | Acceptance | Dependency |
| :--- | :---: | :--- | :--- |
| Tool registry | P0 | Register/query/list tools | None |
| Policy gate | P0 | allow/deny/ask policies take effect | Policy config |
| `read_file` | P0 | Supports path and line ranges; large files auto-truncated | Policy gate |
| `search_files` | P0 | Supports glob/regex with pagination | Policy gate |
| `run_shell` | P0 | Whitelisted commands run directly; others require confirmation | Policy gate |
| **Command injection guard** | P0 | Escape parameters; forbid shell metacharacters | run_shell |
| **Timeout control** | P0 | Each tool call has a timeout (default 60s) | All tools |

### 1.4 Patch Engine

| Requirement | Priority | Acceptance | Dependency |
| :--- | :---: | :--- | :--- |
| Diff generation | P0 | Produce standard unified diff | None |
| Dry-run apply | P0 | Do not modify files; return expected result | Diff generation |
| Apply | P0 | Apply diff with line-offset tolerance | Dry-run |
| Rollback | P0 | Backup before each apply; support rollback | Apply |
| **Binary detection** | P0 | Detect binary files and error instead of diffing | Diff generation |

### 1.5 LLM Integration

| Requirement | Priority | Acceptance | Dependency |
| :--- | :---: | :--- | :--- |
| Provider interface | P0 | Define a unified provider abstraction | None |
| OpenAI adapter | P0 | Supports gpt-4/gpt-3.5-turbo with tool calls | Provider |
| Anthropic adapter | P0 | Supports claude-sonnet-4-20250514 with `tool_use` | Provider |
| Streaming output | P0 | Parse SSE stream correctly; support mid-stream cancel | Provider |
| **Retry strategy** | P0 | Retry retryable errors with exponential backoff | Provider |
| **Rate-limit handling** | P0 | On 429, wait Retry-After before retrying | Provider |

### 1.6 CLI

| Requirement | Priority | Acceptance | Dependency |
| :--- | :---: | :--- | :--- |
| `gm run <prompt>` | P0 | Execute a single task and output the result | Runtime |
| `gm status` | P1 | Show current/recent task status | Store |
| `gm history` | P1 | List past tasks | Store |

### 1.7 Observability (required in Phase 1)

| Requirement | Priority | Acceptance | Dependency |
| :--- | :---: | :--- | :--- |
| **Structured logging** | P0 | Use slog with JSON output | None |
| **Request ID** | P0 | Assign ID per request and propagate through logs | None |
| **Log redaction** | P0 | API keys/sensitive fields automatically redacted | Structured logging |
| Basic metrics | P1 | Requests/latency/error rate | None |

---

## Phase 2: Multi-actor + Interrupt/Resume

**Goal**: Support real-world usage
**Timeline**: 2-4 weeks

### 2.1 Event System

| Requirement | Priority | Acceptance | Dependency |
| :--- | :---: | :--- | :--- |
| Event semantics | P1 | Append/Fork/Preempt/Cancel handled correctly | Runtime |
| Event metadata | P1 | actor/subject/priority populated | Event semantics |
| **Safe preemption** | P1 | Preempt only at safe points; do not break atomic ops | Runtime |

### 2.2 Persistence Enhancements

| Requirement | Priority | Acceptance | Dependency |
| :--- | :---: | :--- | :--- |
| Checkpointing | P1 | Save every N events or T seconds automatically | Store |
| Crash recovery | P1 | Restart from latest checkpoint | Checkpoint |
| **Atomic checkpoints** | P1 | Checkpoint and event log written in one transaction | Store (tx) |
| Outbox pattern | P1 | Events and side effects committed together | Store (tx) |

### 2.3 Sub Agent

| Requirement | Priority | Acceptance | Dependency |
| :--- | :---: | :--- | :--- |
| Task protocol | P1 | Define Task/Result structure | None |
| Sub Agent scheduling | P1 | Main Agent can dispatch subtasks | Task protocol |
| Progress reporting | P1 | Sub Agent reports progress periodically | Sub Agent |
| **Timeout detection** | P1 | Main Agent senses Sub Agent timeout | Sub Agent |
| **Cancel propagation** | P1 | Cancelling Main Agent notifies Sub Agent | Sub Agent |

### 2.4 Context Management

| Requirement | Priority | Acceptance | Dependency |
| :--- | :---: | :--- | :--- |
| Compaction | P1 | Compress history while retaining key info | LLM |
| Token budgeting | P1 | Smart truncation when exceeding budget | LLM |

---

## Phase 3: Solidification & Ecosystem

**Goal**: Accumulated experience and extensibility
**Timeline**: 2-4 weeks

### 3.1 Skill System

| Requirement | Priority | Acceptance | Dependency |
| :--- | :---: | :--- | :--- |
| Skill registry | P2 | Scan and load the `skills/` directory | None |
| Versioning | P2 | Support multiple versions side by side | Skill registry |
| Evaluation cases | P2 | Each skill has test cases | Skill registry |

### 3.2 Scheme Interpreter

| Requirement | Priority | Acceptance | Dependency |
| :--- | :---: | :--- | :--- |
| Scheme parsing | P2 | Parse YAML/JSON scheme files | None |
| Step execution | P2 | Execute steps in defined order | Scheme parsing |
| Error handling | P2 | BLOCKED/FALLBACK modes | Step execution |

### 3.3 Extension Protocols

| Requirement | Priority | Acceptance | Dependency |
| :--- | :---: | :--- | :--- |
| MCP client | P2 | Connect to MCP server and call tools | Tool registry |
| Circuit breaker | P2 | Breaker when MCP server unavailable | MCP client |

### 3.4 Storage Upgrades

| Requirement | Priority | Acceptance | Dependency |
| :--- | :---: | :--- | :--- |
| SQLite adapter | P2 | Implement Store interface | Store interface |
| PostgreSQL adapter | P2 | Implement Store interface | Store interface |
| Migration tool | P2 | Commands to migrate fs -> sqlite -> postgres | All adapters |

---

## Phase 4: Production-ready

**Goal**: Fit enterprise environments
**Timeline**: Ongoing

### 4.1 Observability

| Requirement | Priority | Acceptance | Dependency |
| :--- | :---: | :--- | :--- |
| Tracing | P3 | OpenTelemetry support | None |
| Metrics export | P3 | Prometheus format | None |
| Cost tracking | P3 | Token cost per session | LLM |

### 4.2 Security

| Requirement | Priority | Acceptance | Dependency |
| :--- | :---: | :--- | :--- |
| Permission levels | P3 | Different roles per user | None |
| Audit log | P3 | Trace sensitive operations | None |
| Sensitive redaction | P0 | (Already in Phase 1) | None |

### 4.3 Deployment

| Requirement | Priority | Acceptance | Dependency |
| :--- | :---: | :--- | :--- |
| HTTP API | P3 | Fully exposed RESTful API | Runtime |
| Config hot-reload | P3 | Change config without restart | None |
| Multi-instance | P3 | Coordinate instances (Redis/etcd) | None |

---

## Priority Legend

| Priority | Phase | Meaning |
| :---: | :---: | :--- |
| P0 | 1 | **Required**: product unusable without it |
| P1 | 2 | **Important**: needed in real scenarios |
| P2 | 3 | **Enhancement**: improves UX and extensibility |
| P3 | 4 | **Long-term**: enterprise features |

---

## Risks

| Risk | Impact | Mitigation |
| :--- | :--- | :--- |
| Unstable LLM API | Task interruption | Retry + circuit breaker + fallback |
| Concurrency deadlock | System freeze | Lock timeout + deadlock detection |
| Data loss | Inconsistent state | Atomic writes + Outbox |
| Key leakage | Security incident | Redaction + avoid persistence |
