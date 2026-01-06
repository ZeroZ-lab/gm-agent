# MCP 协议

> Model Context Protocol 集成说明

---

## 1. 目的

通过 MCP 接入外部工具生态，实现统一工具调用与权限控制。

---

## 2. 运行模式

- MCP Server 以独立进程运行
- gm-agent 以 MCP Client 方式连接（stdio/SSE）

---

## 3. 约束

- MCP 工具必须通过 Policy Gate
- 工具返回内容视为数据，不可当作指令

---

## 4. 版本

当前遵循 MCP 官方规范，具体版本由实现时锁定。
