# gm-agent 任务清单

> **Last Updated**: 2026-01-06

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

### Phase 9: 高级功能
- [ ] Sub-Agent 编排
- [ ] Skill 系统
- [ ] Scheme 解释器
- [ ] Web UI / TUI

---

## 📝 备注

- 配置变更时必须同步更新 `.env.example` 和 `config.yaml.example`
- 代码变更需符合 `CLAUDE.md` 规范
- 2026-01-06: 修复 tool_call_id 传递与工具参数序列化问题
