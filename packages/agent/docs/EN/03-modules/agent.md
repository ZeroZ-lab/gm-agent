# Agent Module

> Abstraction for Main Agent and Sub Agents

## Concepts
- **Main Agent**: orchestrates the session, plans steps, and delegates work.
- **Sub Agent**: specialized worker that executes delegated tasks and reports back.
- **Task Protocol**: Task payload + progress/result messages exchanged via events.

## Lifecycle
1. Main Agent receives user goal and creates tasks.
2. Tasks may be delegated to Sub Agents with clear subject and budget.
3. Sub Agents periodically send progress updates; Main Agent can cancel or preempt.
4. Final results are merged into the main context and persisted as artifacts/patches.

## Design Notes
- Agents are stateless functions driven by reducer state.
- Cancellation and timeouts propagate through event metadata.
- Skill/Scheme outputs can seed new Agent prompts for repeatability.
