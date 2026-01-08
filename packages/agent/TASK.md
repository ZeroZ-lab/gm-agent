# gm-agent 任务清单

> **Last Updated**: 2026-01-08 (Phase 1.3 完成)

## ✅ 已完成

### Phase 1: 工程准备
- [x] 创建 `CLAUDE.md` 开发规范
- [x] 初始化 Go Module 和目录结构
- [x] 创建 `Makefile`

### Phase 2: 核心骨架
- [x] 实现 `pkg/types` (Data Models)
- [x] 实现 `pkg/store` (FS Store)
- [x] 实现 `pkg/runtime` (Loop, Reducer, Dispatcher)
- [x] 骨架集成测试

### Phase 3: LLM Gateway
- [x] `pkg/llm/provider.go` (Interfaces)
- [x] `pkg/llm/openai/provider.go` (OpenAI/DeepSeek)
- [x] `pkg/llm/gemini/provider.go` (Native Gemini)
- [x] `pkg/llm/factory/factory.go` (Provider Factory)
- [x] `cmd/gm/main.go` 接入真实 LLM

### Phase 4: Tool System
- [x] Policy Gate (`pkg/tool/policy.go`)
- [x] Built-in Tools (`read_file`, `run_shell`)
- [x] Dynamic Tool Registry

### Phase 5: CLI & 配置
- [x] 加载 `config.yaml` / Env
- [x] 配置重构 (Env 注入 & 安全策略)
- [x] 实现 `gm run [goal]`
- [x] 支持 `.env` 文件
- [x] 添加 `--config` CLI 参数

### Phase 6: 交互工具
- [x] `talk` tool (Stdout)
- [x] `task_complete` tool (退出)
- [x] 验证交互循环

---

## ✅ 已完成 (续)

### Phase 7: 配置重构 (OpenCode Style)
- [x] 重构 `pkg/config` 为 `provider[id]` 结构
- [x] 实现 Provider 自动检测 (Env Vars)
- [x] 更新 `factory.go` 使用新配置
- [x] 更新 `main.go` 适配
- [x] 同步 `.env.example` 和 `config.yaml.example`

---

## 🚧 进行中

*(无)*

### Phase 7: 文档完善
- [ ] 重组 `docs/` 目录结构 (EN/ZH)
- [ ] 补充模块文档

### Phase 8: 测试 & CI
- [ ] 恢复/重写集成测试
- [ ] GitHub Actions CI

### Phase 9: Patch Engine (2026-01-08)
- [x] `pkg/patch/patch.go` - 核心接口和配置
- [x] `pkg/patch/diff.go` - Diff 生成算法 (基于 diffmatchpatch)
- [x] `pkg/patch/apply.go` - Patch 应用逻辑 (含路径验证)
- [x] `pkg/patch/rollback.go` - 回滚机制 (基于备份)
- [x] 添加 `write_file` 工具 (pkg/agent/tools/file_editing.go)
- [x] 添加 `edit_file` 工具 (pkg/agent/tools/file_editing.go)

### Phase 10: 代码库搜索 (2026-01-08)
- [x] `glob` 工具 - 文件模式匹配 (支持 ** 模式)
- [x] `grep` 工具 - 内容搜索 (正则表达式 + 上下文)
- [x] pkg/agent/tools/search.go 实现

### Phase 11: 安全加固 (2026-01-08)
- [x] `pkg/security/validator.go` - 路径验证器
- [x] 路径遍历防护 (../ 检测)
- [x] 可疑模式检测 (SSH keys, credentials, etc.)
- [x] Shell 命令验证器 (危险命令拦截)
- [x] 资源限制定义 (时间/内存/文件大小)

### Phase 12: 高级功能 (规划中)
- [x] 基于 Gin 的 HTTP API（含 OpenAPI 输出）
- [x] 默认 API 模式启动与日志等级配置
- [x] **Phase 1.2 完成 (2026-01-08)**: 工具激活
  - [x] 注册 write_file, edit_file, glob, grep
  - [x] 集成 Patch Engine
  - [x] 测试验证通过
  - 成果: 实际可用性从 35% 提升到 50%
- [x] **Phase 1.3 完成 (2026-01-08)**: Checkpointing UI
  - [x] API: GET /session/:id/checkpoints
  - [x] API: POST /session/:id/rewind
  - [x] CLI: /checkpoints 命令
  - [x] CLI: /rewind <id> 命令
  - [x] Service层实现conversation rewind
  - 成果: 实际可用性从 50% 提升到 58%
- [ ] Sub-Agent 编排
- [ ] Skill 系统
- [ ] Scheme 解释器
- [ ] Web UI / TUI

---

## 📝 备注

- 配置变更时必须同步更新 `.env.example` 和 `config.yaml.example`
- 代码变更需符合 `CLAUDE.md` 规范
- 2026-01-06: 修复 tool_call_id 传递与工具参数序列化问题
- 2026-01-08: **重大更新** - 实现完整的 Patch Engine、代码搜索工具和安全加固
  - 新增 `pkg/patch` 模块 (diff/apply/rollback)
  - 新增 `write_file` 和 `edit_file` 工具
  - 新增 `glob` 和 `grep` 搜索工具
  - 新增 `pkg/security` 安全验证模块
  - 依赖: 添加 `github.com/sergi/go-diff` v1.4.0
  - **⚠️ 关键发现**: 工具已实现但未在 main.go 中注册！实际可用性仅 35%
  - 参考文档: `/docs/GAP-ANALYSIS.md` (已更新为深度对比)

## 🎯 与 Claude Code 的真实差距 (2026-01-08 更新)

经过深入研究 Claude Code 官方能力，发现以下核心差距:

### 已解决 ✅
- ✅ **工具激活** (Phase 1.2): write_file/edit_file/glob/grep 已注册
- ✅ **Checkpointing UI** (Phase 1.3): 用户可以通过CLI和API查看和回滚checkpoint

### 待解决
- ❌ **Plan Mode 缺失**: 无只读分析工作流
- ❌ **Code Rewind**: 仅支持conversation rewind，code rewind待实现

### 成熟度评估
- Phase 1.1 完成: ~35% vs Claude Code
- Phase 1.2 完成: ~50% (+15%)
- **Phase 1.3 完成: ~58% (+8%)**
- Phase 2 完成目标: ~65%

详见: `/docs/GAP-ANALYSIS.md` 和 `/docs/PHASE-1.3-SUMMARY.md`
