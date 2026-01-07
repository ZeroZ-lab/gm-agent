# gm-agent Documentation Center

> **gm-agent** - A Go-based autonomous Agent runtime framework

---

## 📚 Navigation

### 1️⃣ Vision & Requirements
- [Product Vision](./01-vision/README.md) - Why this project exists
- [Feature Requirements](./01-vision/requirements.md) - Phased requirement list

### 2️⃣ System Architecture
- [Architecture Overview](./02-architecture/README.md) - Full system view
- [System Design](./02-architecture/system-design.md) - Core design decisions ⭐

### 3️⃣ Module Design
| Module | Description | Docs |
| :--- | :--- | :--- |
| **Runtime** | State machine + dispatcher + persistence | [Details](./03-modules/runtime.md) |
| **Agent** | Main/Sub Agent abstraction | [Details](./03-modules/agent.md) |
| **Tool** | Tool registration and execution | [Details](./03-modules/tool.md) |
| **LLM** | Multi-model adapter layer | [Details](./03-modules/llm.md) |
| **Patch** | Diff application and rollback | [Details](./03-modules/patch.md) |
| **Store** | Event log and snapshots | [Details](./03-modules/store.md) |
| **Skill** | Persistable reusable capabilities | [Details](./03-modules/skill.md) |
| **Scheme** | Strict procedure interpreter | [Details](./03-modules/scheme.md) |

### 4️⃣ API Definitions
- [CLI Commands](./04-api/cli.md)
- [HTTP API](./04-api/http-api.md)
- [MCP Protocol](./04-api/mcp.md)

### 5️⃣ Deployment & Configuration
- [Installation Guide](./05-deployment/installation.md)
- [Configuration](./05-deployment/configuration.md)

### 6️⃣ Security 🔐
- [Security Overview](./06-security/README.md) - Key management, access control, audit logs ⭐

---

## 🚀 Quick Start

```bash
# Build
go build -o gm ./cmd/gm

# Run
./gm run "Refactor this function for me"
```

---

## 📂 Project Structure

```
gm-agent/
├── cmd/gm/              # CLI entrypoint
├── pkg/
│   ├── runtime/         # Core runtime
│   ├── agent/           # Agent abstraction
│   ├── tool/            # Tooling system
│   ├── llm/             # LLM adapters
│   ├── patch/           # Patch engine
│   ├── store/           # Storage layer
│   ├── skill/           # Skill loader
│   └── scheme/          # Scheme interpreter
├── skills/              # Built-in skills
├── schemes/             # Built-in schemes
└── docs/                # Documentation
```

---

## 🔗 Related Resources

- [OpenCode](https://github.com/anomalyco/opencode) - Design reference
- [MCP Protocol](https://modelcontextprotocol.io/) - Tooling extension standard
