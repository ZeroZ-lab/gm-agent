# CLAUDE.md for gm-agent

> **AI Context & Developer Guide**
> This file contains essential commands, standards, and structural information for working on the gm-agent project.

---

## 🛠 Common Commands

- **Build**: `make build` (Output: `bin/gm`)
- **Run**: `make run`
- **Test (All)**: `make test`
- **Test (Unit)**: `make test-unit`
- **Test (Integration)**: `make test-integration`
- **Lint**: `make lint`
- **Verify (Pre-commit)**: `make verify`
- **Clean**: `make clean`
- **Generate API Docs**: `swag init -g cmd/gm/main.go -o docs/api` (Planned)

---

## 🏗 Project Structure

```
gm-agent/
├── cmd/
│   └── gm/              # Main CLI entry point
├── pkg/
│   ├── runtime/         # Core loop, Reducer, Dispatcher
│   ├── store/           # Persistence (FS, SQLite), Models (State, Event)
│   ├── llm/             # LLM Gateway, Providers, Circuit Breaker
│   ├── tool/            # Tool Registry, Policy Gate, Sandbox
│   ├── patch/           # Diff Engine, Apply, Rollback
│   └── agent/           # Sub-agent orchestration
├── docs/                # Design Docs (Single Source of Truth)
│   ├── 01-vision/       # Requirements & Vision
│   ├── 02-architecture/ # System Design & Data Models
│   ├── 03-modules/      # Detailed Module Designs
│   └── 06-security/     # Security Architecture
└── tools/               # Build scripts & helpers
```

---

## 📚 Documentation Map (Single Source of Truth)

| Domain | File | Description |
| :--- | :--- | :--- |
| **Data Model** | `docs/02-architecture/data-model.md` | **Authority**. Definitions of `State`, `Event`, `Task`, `Goal`, `Context`. |
| **System Design** | `docs/02-architecture/system-design.md` | Core architecture decisions (Event-Driven, Reducer, Checkpoints). |
| **Runtime** | `docs/03-modules/runtime.md` | Main loop pseudo-code, error handling table, recovery logic. |
| **Agent Protocol** | `docs/03-modules/agent.md` | Task dispatch protocols, Sub-Agent isolation strategies. |
| **Store** | `docs/03-modules/store.md` | Storage interfaces, FS/DB implementations, atomic write guarantees. |
| **LLM Gateway** | `docs/03-modules/llm.md` | Provider adapters, circuit breaker state machine, token management. |
| **Tools** | `docs/03-modules/tool.md` | Tool registry, Policy Gate configuration, execution sandbox. |
| **Patch Engine** | `docs/03-modules/patch.md` | Diff generation algorithm, patch application, rollback mechanism. |
| **Security** | `docs/06-security/README.md` | Threat model, secret management, input validation rules. |
| **Requirements** | `docs/01-vision/requirements.md` | MVP acceptance criteria and phased roadmap. |

> ⚠️ Always refer to `data-model.md` for struct definitions. Do not infer from code if it contradicts this doc.

---

## 🧩 Architecture Principles

1.  **Event-Driven**: State changes ONLY via Events.
    -   Pattern: `(State, Event) -> (NewState, Commands)`
    -   **NO** direct state mutation outside Reducer.
2.  **Durability**:
    -   Events are immutable and append-only.
    -   State is a snapshot of event history.
    -   **MUST** use atomic writes (write-temp + rename) for file operations.
3.  **Safety**:
    -   **Policy Gate** enforces allow/ask/deny logic for all tools.
    -   **Context** propagation is mandatory for cancellation and timeouts.
4.  **Error Handling**:
    -   Classify errors: `Retryable`, `Recoverable` (ask LLM), `Fatal`.
    -   Use `fmt.Errorf("%w", err)` for wrapping.

---

## 🧙‍♂️ Go Expert Best Practices

### 1. Code Construction
-   **Interface Design**:
    -   **Accept Interfaces, Return Structs**: Decouple implementation from consumption.
    -   **Define Client-Side**: Interfaces should be defined where they are used, not where they are implemented.
    -   **Keep Small**: Single-method interfaces (e.g., `Reader`, `Runner`) are more powerful.
-   **Error Handling**:
    -   **Wrap, Don't Hide**: Use `fmt.Errorf("context: %w", err)` to add context while preserving the underlying error.
    -   **Sentinel Errors**: Use `errors.Is` for value comparisons (e.g., `ErrNotFound`).
    -   **No Panic**: Handle errors explicitly. Panic only on startup or invariant violation.
-   **Concurrency**:
    -   **Context is King**: Pass `context.Context` as the first arg modules. Strictly respect cancellation.
    -   **No Leaks**: Never start a goroutine without a clear exit strategy (channel or context).
    -   **Structured Concurrency**: Prefer `errgroup` over raw `go func()` + `WaitGroup` for task groups.

### 2. Testing Strategy
-   **Strict Rule**: Every new function must have a corresponding unit test.
-   **Table-Driven**: Use table-driven tests for all logic. It's the Go standard.
-   **Black-Box Testing**: Use `package pkg_test` to test public APIs. Avoid testing internal implementation details.
-   **Race Detection**: CI must run with `-race`.
-   **Golden Files**: For complex outputs (like JSON or large text), use golden files instead of inline strings.

### 3. Performance & Optimization
-   **Pre-allocation**: `make([]T, 0, cap)` allows avoiding re-allocations in loops.
-   **String Building**: Use `strings.Builder`, never `+` in loops.
-   **Pointers**: Don't use pointers for small structs just to "save memory" (escape analysis may cause heap allocation). Use pointers for semantics (shared access/mutation), not strictly optimization.

### 4. Project Layout Authority
-   `cmd/`: Main entry points. Keep minimal.
-   `internal/`: Private application code. Enforces boundaries.
-   `pkg/`: Library code ok for external consumption.
-   `tests/`: Integration/E2E tests.

---

## 🛠 Development Workflow

1.  **Thinking Phase (Strict)**:
    -   **Documentation First**: Before ANY code implementation or modification, check `docs/`.
    -   **Update Docs**: If the design is outdated, update the `docs/` FIRST. Code must match the docs.
    -   Update `docs/02-architecture/data-model.md` if data structures change.
    -   Verify module boundaries in `system-design.md`.
2.  **Coding Phase**:
    -   Define Interface -> Write Test (Red) -> Implementation (Green).
    -   **Configuration Sync**: If you add or modify a configuration field, you **MUST** update `.env.example` and `config.yaml.example` immediately.
    -   Run `golangci-lint` locally before committing.
3.  **Review Phase**:
    -   Check "Security Checklist".
    -   Verify test coverage for critical paths (Runtime/Store).

---

## � Collaboration Philosophy

> **"Slow is Smooth, Smooth is Fast."**

-   **Step-by-Step**: Do not rush. Implement one logical unit at a time.
-   **No Hallucinated Complexity**: Solve the problem at hand, don't over-engineer for hypothetical futures (YAGNI).
-   **Verify First**: Never assume code works. Run verify commands after every significant change.
-   **Atomic Moves**: Make small, reversible changes.

---

## �🔐 Security Checklist

-   [ ] **Secrets**: Never commit secrets. Load from Env/Keychain.
-   [ ] **Input**: Validate all inputs at module boundaries.
-   [ ] **Path**: Sanitize file paths to prevent traversal (`../../`).
-   [ ] **Shell**: Use `exec.Command` with arguments, NEVER `bash -c` with unsanitized string concatenation.
