# CLI 命令

> gm-agent 命令行接口定义

---

## 1. 约定

- 命令格式：`gm <command> [args] [flags]`
- 统一输出为结构化日志（可选 `--json`）

---

## 2. 核心命令

### 2.1 `gm run`

执行单次任务。

```bash
gm run "<prompt>" [--model <model>] [--max-steps <n>]
```

### 2.2 `gm status`

查看当前/最近任务状态。

```bash
gm status [--latest]
```

### 2.3 `gm history`

列出历史任务。

```bash
gm history [--limit <n>]
```

---

## 3. 运维类命令

### 3.1 `gm store migrate`

存储后端迁移。

```bash
gm store migrate --from fs --to sqlite
```

---

## 4. 退出码

- `0`: 成功
- `1`: 运行时错误
- `2`: 参数错误
