# gm-agent 文档中心

> **gm-agent** - 基于 Go 语言的自主 Agent Runtime 框架

---

## 📚 文档导航

### 1️⃣ 愿景与需求
- [产品愿景](./01-vision/README.md) - 为什么要做这个项目
- [功能需求](./01-vision/requirements.md) - 分阶段需求列表

### 2️⃣ 系统架构
- [架构概览](./02-architecture/README.md) - 系统全景图
- [系统设计](./02-architecture/system-design.md) - 核心设计决策 ⭐

### 3️⃣ 模块设计
| 模块 | 描述 | 文档 |
| :--- | :--- | :--- |
| **Runtime** | 状态机 + 调度器 + 持久化 | [详情](./03-modules/runtime.md) |
| **Agent** | Main/Sub Agent 抽象 | [详情](./03-modules/agent.md) |
| **Tool** | 工具注册与执行 | [详情](./03-modules/tool.md) |
| **LLM** | 多模型适配层 | [详情](./03-modules/llm.md) |
| **Patch** | Diff 应用与回滚 | [详情](./03-modules/patch.md) |
| **Store** | 事件日志与快照 | [详情](./03-modules/store.md) |
| **Skill** | 可复用能力落盘 | [详情](./03-modules/skill.md) |
| **Scheme** | 严格流程解释器 | [详情](./03-modules/scheme.md) |

### 4️⃣ 接口定义
- [CLI 命令](./04-api/cli.md)
- [HTTP API](./04-api/http-api.md)
- [MCP 协议](./04-api/mcp.md)

### 5️⃣ 部署与配置
- [安装指南](./05-deployment/installation.md)
- [配置说明](./05-deployment/configuration.md)

### 6️⃣ 安全架构 🔐
- [安全总览](./06-security/README.md) - 密钥管理、权限控制、审计日志 ⭐

---

## 🚀 快速开始

```bash
# 构建
go build -o gm ./cmd/gm

# 运行
./gm run "帮我重构这个函数"
```

---

## 📂 项目结构

```
gm-agent/
├── cmd/gm/              # CLI 入口
├── pkg/
│   ├── runtime/         # 核心 Runtime
│   ├── agent/           # Agent 抽象
│   ├── tool/            # 工具系统
│   ├── llm/             # LLM 适配
│   ├── patch/           # Patch Engine
│   ├── store/           # 存储层
│   ├── skill/           # Skill 加载
│   └── scheme/          # Scheme 解释器
├── skills/              # 内置 Skills
├── schemes/             # 内置 Schemes
└── docs/                # 本文档
```

---

## 🔗 相关资源

- [OpenCode](https://github.com/anomalyco/opencode) - 设计参考
- [MCP 协议](https://modelcontextprotocol.io/) - 工具扩展标准
