# Tool Module

> Tool registration, policy enforcement, and execution

## Registry
- Register tools with name, description, input/output schema, and executor.
- Query/list tools for LLM tool-call exposure.

## Policy
- Three-level policy: `allow`, `deny`, `ask`.
- Policies evaluated per actor/subject; defaults configurable.
- Sensitive operations (filesystem, shell) require explicit allowlist or confirmation.

## Executors
- Built-in executors: `read_file`, `search_files`, `run_shell`, HTTP/web fetchers.
- Enforce timeouts (default 60s) and command-injection mitigation (argument escaping, forbid shell metacharacters).
- Results include structured payload + stderr/stdout logs.

## Auditing
- Every tool invocation is logged as an event with correlation IDs.
- Artifacts from tools (files, search hits) can be persisted through the store.
