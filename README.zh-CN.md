# gm-agent

企业级可扩展 Agent 运行时与服务端框架，提供稳定的事件驱动内核、可插拔工具系统与 LLM 网关，并通过统一的 HTTP API 对外提供能力。

Read this document in English: `README.md`

## 核心能力

- Durable Runtime（Reducer + Dispatcher + Checkpoint）
- LLM 网关与多 Provider 适配
- 工具注册、策略控制与隔离执行
- 会话管理与事件存储（FS Store）
- 标准化 HTTP API（含 Swagger 文档）

## 架构概览

详细架构与设计说明见：
- `packages/agent/docs/ZH/02-architecture/README.md`
- `packages/agent/docs/ZH/02-architecture/system-design.md`

## 快速开始

环境要求：Go 1.25.5

```bash
make build
make run
```

默认 HTTP API 监听 `:8080`。

## 配置

- 示例配置：`packages/agent/config.yaml.example`
- 配置优先级：环境变量（`GM_` 前缀） > 配置文件 > 默认值
- 可配置项包括：HTTP 监听地址、API Key、安全策略、LLM Provider 等

## API 文档

- OpenAPI：`packages/agent/docs/swagger.yaml`
- Swagger JSON：`packages/agent/docs/swagger.json`

## 目录结构（单仓多应用）

```
packages/
  agent/      # 核心服务与运行时
  web/        # Web 客户端
  tui/        # TUI 客户端
  desktop/    # 桌面端
```

## 运行与测试

```bash
make test
make lint
make verify
```

## 安全与访问控制

支持通过配置启用/限制工具调用、文件系统访问与网络访问，并可为 HTTP API 设置访问密钥。详见配置文件与安全模块文档。
