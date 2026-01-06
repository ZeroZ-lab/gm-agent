# 功能需求

> 分阶段的功能需求列表（含验收标准）

---

## Phase 1: MVP（最小可行产品）

**目标**: 单 Agent 跑通完整链路  
**预计时间**: 2-4 周

### 1.1 Runtime Core

| 需求 | 优先级 | 验收标准 | 依赖 |
| :--- | :---: | :--- | :--- |
| Reducer 模式 | P0 | `(state, event) -> (state, commands)` 纯函数测试通过 | 无 |
| Dispatcher | P0 | 能分发到 LLM/Tool/Patch 执行器 | Reducer |
| Worker Pool | P0 | 支持配置最大并发数，goroutine 无泄漏 | 无 |
| Subject Lock | P0 | 同一资源写操作串行，读操作并行 | 无 |
| **死锁预防** | P0 | 锁超时机制，获取锁超过 30s 自动失败 | Subject Lock |

### 1.2 存储

| 需求 | 优先级 | 验收标准 | 依赖 |
| :--- | :---: | :--- | :--- |
| FS Store | P0 | events.jsonl 追加写入，state.json 覆盖写入 | 无 |
| Artifacts 管理 | P0 | 支持保存/读取/列出/删除 | FS Store |
| 幂等 event_id | P0 | 相同 event_id 不重复写入 | FS Store |
| **原子写入** | P0 | 写入失败不留下半成品文件 (write-rename) | FS Store |

### 1.3 工具系统

| 需求 | 优先级 | 验收标准 | 依赖 |
| :--- | :---: | :--- | :--- |
| Tool Registry | P0 | 支持注册/查询/列出工具 | 无 |
| Policy Gate | P0 | allow/deny/ask 三级策略生效 | Policy 配置 |
| `read_file` | P0 | 支持路径、行范围，大文件自动截断 | Policy Gate |
| `search_files` | P0 | 支持 glob/regex，结果分页 | Policy Gate |
| `run_shell` | P0 | 白名单命令直接执行，其他需确认 | Policy Gate |
| **命令注入防护** | P0 | 参数转义，禁止 shell 元字符 | run_shell |
| **超时控制** | P0 | 每个工具调用有超时限制 (默认 60s) | 所有工具 |

### 1.4 Patch Engine

| 需求 | 优先级 | 验收标准 | 依赖 |
| :--- | :---: | :--- | :--- |
| Diff 生成 | P0 | 生成标准 unified diff 格式 | 无 |
| Dry-run Apply | P0 | 不实际修改文件，返回预期结果 | Diff 生成 |
| Apply | P0 | 实际应用 diff，支持行号偏移容错 | Dry-run |
| Rollback | P0 | 每次 Apply 前备份，支持回滚 | Apply |
| **二进制检测** | P0 | 检测到二进制文件返回错误，不尝试 diff | Diff 生成 |

### 1.5 LLM 集成

| 需求 | 优先级 | 验收标准 | 依赖 |
| :--- | :---: | :--- | :--- |
| Provider 接口 | P0 | 定义统一的 Provider 抽象 | 无 |
| OpenAI 适配 | P0 | 支持 gpt-4/gpt-3.5-turbo，工具调用 | Provider |
| Anthropic 适配 | P0 | 支持 claude-sonnet-4-20250514，tool_use | Provider |
| 流式输出 | P0 | SSE 流正确解析，支持中途取消 | Provider |
| **重试策略** | P0 | 可重试错误自动重试，指数退避 | Provider |
| **速率限制处理** | P0 | 429 错误等待 Retry-After 后重试 | Provider |

### 1.6 CLI

| 需求 | 优先级 | 验收标准 | 依赖 |
| :--- | :---: | :--- | :--- |
| `gm run <prompt>` | P0 | 执行单次任务，输出结果 | Runtime |
| `gm status` | P1 | 显示当前/最近任务状态 | Store |
| `gm history` | P1 | 列出历史任务 | Store |

### 1.7 可观测性 (Phase 1 必须)

| 需求 | 优先级 | 验收标准 | 依赖 |
| :--- | :---: | :--- | :--- |
| **结构化日志** | P0 | 使用 slog，JSON 格式输出 | 无 |
| **Request ID** | P0 | 每次请求分配 ID，贯穿日志 | 无 |
| **日志脱敏** | P0 | API Key/敏感信息自动脱敏 | 结构化日志 |
| 基础 Metrics | P1 | 请求数/延迟/错误率 | 无 |

---

## Phase 2: 多请求者 + 中断恢复

**目标**: 支持真实使用场景  
**预计时间**: 2-4 周

### 2.1 事件系统

| 需求 | 优先级 | 验收标准 | 依赖 |
| :--- | :---: | :--- | :--- |
| 事件归类 | P1 | Append/Fork/Preempt/Cancel 四种语义正确处理 | Runtime |
| Event 元数据 | P1 | actor/subject/priority 字段完整 | 事件归类 |
| **抢占安全点** | P1 | Preempt 只在安全点生效，不中断原子操作 | Runtime |

### 2.2 持久化增强

| 需求 | 优先级 | 验收标准 | 依赖 |
| :--- | :---: | :--- | :--- |
| Checkpoint | P1 | 每 N 个事件或 T 秒自动保存 | Store |
| Crash Recovery | P1 | 重启后从最近 checkpoint 恢复 | Checkpoint |
| **原子 Checkpoint** | P1 | checkpoint 和事件日志同事务写入 | Store (事务) |
| Outbox 模式 | P1 | 事件和副作用同事务写入 | Store (事务) |

### 2.3 Sub Agent

| 需求 | 优先级 | 验收标准 | 依赖 |
| :--- | :---: | :--- | :--- |
| Task Protocol | P1 | 定义 Task/Result 结构 | 无 |
| Sub Agent 调度 | P1 | Main Agent 能派发子任务 | Task Protocol |
| 进度上报 | P1 | Sub Agent 周期性上报进度 | Sub Agent |
| **超时检测** | P1 | Sub Agent 超时 Main Agent 能感知 | Sub Agent |
| **取消传播** | P1 | Main Agent 取消时 Sub Agent 收到通知 | Sub Agent |

### 2.4 上下文管理

| 需求 | 优先级 | 验收标准 | 依赖 |
| :--- | :---: | :--- | :--- |
| Compaction | P1 | 压缩历史消息，保留关键信息 | LLM |
| Token 预算 | P1 | 超出预算智能截断 | LLM |

---

## Phase 3: 固化与生态

**目标**: 经验可积累，能力可扩展  
**预计时间**: 2-4 周

### 3.1 Skill 系统

| 需求 | 优先级 | 验收标准 | 依赖 |
| :--- | :---: | :--- | :--- |
| Skill Registry | P2 | 扫描并加载 skills/ 目录 | 无 |
| 版本管理 | P2 | 支持 skill 多版本共存 | Skill Registry |
| 评估用例 | P2 | 每个 skill 有测试用例 | Skill Registry |

### 3.2 Scheme 解释器

| 需求 | 优先级 | 验收标准 | 依赖 |
| :--- | :---: | :--- | :--- |
| Scheme 解析 | P2 | 解析 YAML/JSON scheme 文件 | 无 |
| 步骤执行 | P2 | 按定义顺序执行步骤 | Scheme 解析 |
| 异常处理 | P2 | BLOCKED/FALLBACK 两种模式 | 步骤执行 |

### 3.3 扩展协议

| 需求 | 优先级 | 验收标准 | 依赖 |
| :--- | :---: | :--- | :--- |
| MCP Client | P2 | 连接 MCP Server，调用工具 | Tool Registry |
| 熔断策略 | P2 | MCP Server 不可用时熔断 | MCP Client |

### 3.4 存储升级

| 需求 | 优先级 | 验收标准 | 依赖 |
| :--- | :---: | :--- | :--- |
| SQLite 适配 | P2 | 实现 Store 接口 | Store 接口 |
| PostgreSQL 适配 | P2 | 实现 Store 接口 | Store 接口 |
| 迁移工具 | P2 | fs -> sqlite -> postgres 迁移命令 | 所有适配 |

---

## Phase 4: 生产就绪

**目标**: 可用于企业环境  
**预计时间**: 持续迭代

### 4.1 可观测性

| 需求 | 优先级 | 验收标准 | 依赖 |
| :--- | :---: | :--- | :--- |
| Trace 链路 | P3 | 支持 OpenTelemetry | 无 |
| Metrics 导出 | P3 | Prometheus 格式导出 | 无 |
| 成本统计 | P3 | 按 session 统计 token 消耗 | LLM |

### 4.2 安全

| 需求 | 优先级 | 验收标准 | 依赖 |
| :--- | :---: | :--- | :--- |
| 权限分级 | P3 | 多用户不同权限 | 无 |
| 审计日志 | P3 | 敏感操作留痕 | 无 |
| 敏感信息脱敏 | P0 | (已在 Phase 1) | 无 |

### 4.3 部署

| 需求 | 优先级 | 验收标准 | 依赖 |
| :--- | :---: | :--- | :--- |
| HTTP API | P3 | RESTful API 完整暴露 | Runtime |
| 配置热加载 | P3 | 修改配置无需重启 | 无 |
| 多实例支持 | P3 | 多实例协调 (Redis/etcd) | 无 |

---

## 优先级说明

| 优先级 | Phase | 说明 |
| :---: | :---: | :--- |
| P0 | 1 | **必须**：没有这个功能无法使用 |
| P1 | 2 | **重要**：真实场景需要 |
| P2 | 3 | **优化**：提升体验和扩展性 |
| P3 | 4 | **长期**：企业级特性 |

---

## 风险项

| 风险 | 影响 | 缓解措施 |
| :--- | :--- | :--- |
| LLM API 不稳定 | 任务中断 | 重试 + 熔断 + 降级 |
| 并发死锁 | 系统卡死 | 锁超时 + 死锁检测 |
| 数据丢失 | 状态不一致 | 原子写入 + Outbox |
| 密钥泄露 | 安全事故 | 脱敏 + 不落盘 |
