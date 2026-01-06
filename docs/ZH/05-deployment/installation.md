# 安装指南

> gm-agent 安装与启动

---

## 1. 构建

```bash
go build -o gm ./cmd/gm
```

---

## 2. 运行

```bash
./gm run "帮我重构这个函数"
```

---

## 3. 启用 Git Hooks（提交前自动 lint）

本仓库提供了预设的 `pre-commit` 钩子，提交前会自动执行 `make lint`：

```bash
make hooks
```

> 首次克隆仓库后执行一次即可。钩子依赖 `golangci-lint`，如未安装请参考 https://golangci-lint.run/usage/install/ 。

---

## 4. 目录说明

- 数据目录默认：`~/.gm-agent/`
- 日志输出默认：stdout
