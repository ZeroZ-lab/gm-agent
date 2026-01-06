# Product Vision

> **gm-agent** - A durable execution engine for autonomous Agents

---

## 🎯 Core Positioning

**gm-agent is not a chatbot; it is an autonomous execution runtime.**

When AI workloads need more than single-turn responses and must:
- Run tasks that last minutes or hours
- React to external events (user/file/timer/webhook)
- Accept concurrent requests from multiple initiators
- Be interruptible, recoverable, auditable, and reversible

You need an **event-driven durable runtime**, not an endless while-loop.

---

## 💡 Key Ideas

### Autonomy ≠ Lack of Structure

| Misconception | Correct Understanding |
| :--- | :--- |
| Let the LLM decide all control flow | Control flow lives in the runtime; the LLM only makes decisions |
| Draw DAGs externally | Keep a tiny control kernel inside |
| Unlimited freedom | Freedom within guardrails |

### Guiding Principles

1. **Control flow is in the runtime, not in the model.**
2. **Side effects go through Jobs; mutations go through Patches and must be reversible.**
3. **Produce suggestions concurrently but serialize writes (single-writer, multi-reader).**
4. **Any long task needs safe points, interruptibility, and recovery.**
5. **Successful flows can be solidified as Skills; strict flows can be solidified as Schemes.**

---

## 🆚 How We Differ

| Dimension | LangChain/LlamaIndex | Dify/Coze | gm-agent |
| :--- | :--- | :--- | :--- |
| **Control Flow** | Code-defined | Visual DAG | Event-driven state machine |
| **Durability** | None / DIY | Platform-managed | Durable (built-in) |
| **Recovery** | None | Depends on platform | Checkpoints + replay |
| **Language** | Python | No-code | **Go** |
| **Deployment** | Self-hosted | SaaS | **Local-first** |

---

## 🎨 Use Cases

### Phase 1: Personal Productivity
- Code refactoring assistant
- Documentation assistant
- Knowledge organization assistant

### Phase 2: Team Collaboration
- Multi-person task assignment
- Automated code review
- Documentation pipelines

### Phase 3: Enterprise
- CI/CD integrations
- Multi-Agent collaboration
- Audit and compliance

---

## 🏆 Vision

> A truly autonomous system should look like a transit system:
> - Clear rules (policies, locks, budgets)
> - Interpretable traces
> - Recoverable and durable
> - Accumulative knowledge (Skills/Schemes)
>
> Within these boundaries, Agents can be free without losing control.

---

## 📖 Next Steps

- [Feature Requirements](./requirements.md) - Phased requirement list
- [System Architecture](../02-architecture/README.md) - Architecture overview
