# HTTP API

> 基于 Gin 的 REST 接口，提供 CS 架构入口

---

## 1. 基础约定

- Base URL: `/api/v1`
- 认证：`X-API-Key` 头（可选，关闭则匿名访问）
- 请求/响应：JSON，统一 envelop（`data` / `error`）风格与 OpenCode 对齐

---

## 2. 端点

- `POST /api/v1/sessions`：创建新会话，传入 `prompt` 作为初始目标。
- `GET /api/v1/sessions/{id}`：查询会话状态与最新状态快照。
- `GET /api/v1/sessions/{id}/events`：查看事件流，支持 `after` 游标。
- `POST /api/v1/sessions/{id}/cancel`：请求取消/打断运行。

### OpenAPI
- `GET /api/openapi.json`：OpenAPI 3.0 规格文档，便于 SDK/前端生成。

### 健康检查
- `GET /healthz`：存活探针。

---

## 3. 错误格式

```json
{
  "error": "invalid api key"
}
```
