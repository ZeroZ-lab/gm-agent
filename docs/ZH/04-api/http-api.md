# HTTP API

> RESTful API（Phase 4）

---

## 1. 范围

MVP 阶段不提供 HTTP API。此文档用于预留接口规范。

---

## 2. 基础约定

- Base URL: `/api/v1`
- 认证：Bearer Token
- 请求/响应：JSON

---

## 3. 预留端点

### 3.1 健康检查

```
GET /api/v1/health
```

### 3.2 任务执行

```
POST /api/v1/tasks
```

请求体：

```json
{
  "prompt": "重构这个函数",
  "model": "openai/gpt-4"
}
```

---

## 4. 错误格式

```json
{
  "error": {
    "code": "INVALID_ARGUMENT",
    "message": "missing prompt"
  }
}
```
