# 模块设计: Scheme

> 严格流程解释器与可执行规范

---

## 1. 概述

Scheme 用于把“严格的流程步骤”固化为可执行描述，适合高风险或高一致性要求的任务。

核心目标：
- **低自由度**：减少模型的自由发挥
- **可重复**：相同输入得到可预期结果
- **可审计**：每一步有迹可循

---

## 2. Scheme 文件格式

支持 YAML/JSON，最小结构：

```yaml
name: doc-review
version: 1
steps:
  - id: read
    type: tool
    tool: read_file
    inputs:
      path: "{{doc_path}}"
  - id: analyze
    type: llm
    prompt: "请按规范输出问题清单"
  - id: output
    type: write
    target: "stdout"
```

---

## 3. 解释器接口

```go
type SchemeInterpreter interface {
    Execute(ctx context.Context, scheme *Scheme, inputs map[string]any) (*SchemeResult, error)
}

type Scheme struct {
    Name    string
    Version int
    Steps   []SchemeStep
}

type SchemeStep struct {
    ID     string
    Type   string   // tool | llm | write | branch
    Tool   string   // when Type=tool
    Prompt string   // when Type=llm
    Inputs map[string]any
}

type SchemeResult struct {
    Status string
    Output string
    Steps  []StepResult
}
```

---

## 4. 错误处理

- **BLOCKED**：缺少输入或权限不足，立即停止
- **FALLBACK**：可回退到普通 LLM 模式

---

## 5. 已知限制

- MVP 阶段仅支持线性流程
- 条件分支与循环在 Phase 2 规划
