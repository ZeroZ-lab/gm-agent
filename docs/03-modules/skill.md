# 模块设计: Skill

> 可复用能力落盘与加载

---

## 1. 概述

Skill 用于把“可复用的能力与流程”固化到磁盘，供 Runtime 在任务中加载并使用。

核心目标：
- **可积累**：把高频成功路径沉淀为标准能力
- **可复用**：跨任务、跨项目复用
- **可审计**：加载来源可追踪

---

## 2. 目录结构

```
skills/
└── <skill-name>/
    └── SKILL.md
```

**SKILL.md 约束：**
- 必须包含 YAML frontmatter
- 必须包含 `name` 和 `description`
- `description` 作为触发描述，用于匹配加载时机

示例：

```yaml
---
name: review-software-docs
description: 评审软件项目文档并输出问题清单
---
```

---

## 3. Skill Registry

```go
// SkillRegistry 管理技能加载与查询
type SkillRegistry interface {
    // 启动时扫描并加载
    LoadAll(root string) error

    // 查询单个技能
    Get(name string) (*Skill, bool)

    // 列出所有技能
    List() []Skill
}

type Skill struct {
    Name        string
    Description string
    Path        string
    Content     string
}
```

---

## 4. 触发与加载

触发原则：
- 基于用户请求与 Skill `description` 的匹配
- 由 Runtime/Agent 决定是否加载

加载策略：
- 仅加载匹配的 Skill（避免上下文污染）
- 加载记录写入审计日志

---

## 5. 安全边界

- 仅允许加载 `skills/` 目录下的技能
- 禁止加载外部路径或网络资源
- 加载失败不影响主流程（降级为无 Skill）

---

## 6. 已知限制

- MVP 阶段不支持热更新（仅启动时加载）
- Skill 不支持版本并存（Phase 2 规划）
