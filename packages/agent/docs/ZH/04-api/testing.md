# API 测试指南

本文档介绍如何验证 GM-Agent 的 HTTP API 接口。

## 1. 自动化单元测试

项目包含完整的 Handler 和 Service 层测试，位于 `pkg/api/server_test.go`。

运行测试：
```bash
# 在 packages/agent 目录下
go test -v ./pkg/api/...
```

预期输出：
```
=== RUN   TestCreateSessionAndStatus
--- PASS: TestCreateSessionAndStatus (0.01s)
=== RUN   TestListSessions
--- PASS: TestListSessions (0.00s)
...
PASS
ok      github.com/gm-agent-org/gm-agent/pkg/api    0.554s
```

## 2. 手动测试 (Curl)

启动服务器（开启开发模式以支持 Swagger）：
```bash
# 在 packages/agent 目录下
export GM_DEV_MODE=true
go run cmd/gm/main.go
```

### 2.1 健康检查
```bash
curl http://localhost:8080/health
# {"status":"healthy","version":"1.0.0"}
```

### 2.2 创建会话
```bash
curl -X POST http://localhost:8080/api/v1/session \
  -H "Content-Type: application/json" \
  -d '{"prompt": "你好，请帮我写一个 Hello World"}'
# {"id":"ses_01...","status":"running",...}
```

### 2.3 监听 SSE 事件流
**注意**：需要在另一个终端执行，替换 `<session_id>` 为上一步返回的 ID。
```bash
curl -N http://localhost:8080/api/v1/session/<session_id>/event
# event: connected
# data: {"session_id":"..."}
# ...
```

### 2.4 查看会话列表
```bash
curl http://localhost:8080/api/v1/session
```

### 2.5 发送消息 (交互)
```bash
curl -X POST http://localhost:8080/api/v1/session/<session_id>/message \
  -H "Content-Type: application/json" \
  -d '{"content": "请把密码改长一点", "semantic": "append"}'
# {"id":"...","status":"running",...}
```

### 2.6 取消会话
```bash
curl -X POST http://localhost:8080/api/v1/session/<session_id>/cancel
```

## 3. Swagger UI 可视化测试

在 `GM_DEV_MODE=true` 模式下，访问：
👉 **http://localhost:8080/swagger/index.html**

你可以在页面上直接点击 "Try it out" 发送请求并查看响应。
