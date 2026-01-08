# API Testing Guide

This document describes how to verify the GM-Agent HTTP API interfaces.

## 1. Automated Unit Tests

The project contains full Handler and Service layer tests located in `pkg/api/server_test.go`.

Run tests:
```bash
# In packages/agent directory
go test -v ./pkg/api/...
```

Expected output:
```
=== RUN   TestCreateSessionAndStatus
--- PASS: TestCreateSessionAndStatus (0.01s)
=== RUN   TestListSessions
--- PASS: TestListSessions (0.00s)
...
PASS
ok      github.com/gm-agent-org/gm-agent/pkg/api    0.554s
```

## 2. Manual Testing (Curl)

Start the server (enable dev mode for Swagger support):
```bash
# In packages/agent directory
export GM_DEV_MODE=true
go run cmd/gm/main.go
```

### 2.1 Health Check
```bash
curl http://localhost:8080/health
# {"status":"healthy","version":"1.0.0"}
```

### 2.2 Create Session
```bash
curl -X POST http://localhost:8080/api/v1/sessions \
  -H "Content-Type: application/json" \
  -d '{"prompt": "Hello, please write a Hello World for me"}'
# {"id":"ses_01...","status":"running",...}
```

### 2.3 Listen to SSE Event Stream
**Note**: Run this in a separate terminal, replacing `<session_id>` with the ID returned above.
```bash
curl -N http://localhost:8080/api/v1/sessions/<session_id>/events
# event: connected
# data: {"session_id":"..."}
# ...
```

### 2.4 List Sessions
```bash
curl http://localhost:8080/api/v1/sessions
```

### 2.5 Send Message (Interaction)
```bash
curl -X POST http://localhost:8080/api/v1/sessions/<session_id>/messages \
  -H "Content-Type: application/json" \
  -d '{"content": "Please make the password longer", "semantic": "append"}'
# {"id":"...","status":"running",...}
```

### 2.6 Cancel Session
```bash
curl -X POST http://localhost:8080/api/v1/sessions/<session_id>/cancel
```

## 3. Swagger UI Visualization

In `GM_DEV_MODE=true` mode, visit:
👉 **http://localhost:8080/swagger/index.html**

You can click "Try it out" directly on the page to send requests and view responses.
