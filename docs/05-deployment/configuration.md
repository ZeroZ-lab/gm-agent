# 配置说明

> gm-agent 配置示例与字段说明

---

## 1. 配置文件位置

默认读取 `config.yaml`，也可通过环境变量指定。

---

## 2. 示例配置

```yaml
runtime:
  max_steps: 100
  checkpoint_interval: 10

store:
  backend: fs
  fs:
    base_path: ~/.gm-agent/data

llm:
  default_model: "openai/gpt-4"
  providers:
    openai:
      api_key: "${OPENAI_API_KEY}"

policy:
  tools:
    default: ask
```

---

## 3. 环境变量

- `OPENAI_API_KEY`
- `ANTHROPIC_API_KEY`
