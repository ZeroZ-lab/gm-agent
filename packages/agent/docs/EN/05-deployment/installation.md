# Installation Guide

## Prerequisites
- Go toolchain (1.22+ recommended)
- Access to LLM provider keys (OpenAI/Anthropic)

## Build
```bash
go build -o gm ./cmd/gm
```

## Configuration
- Copy `config.yaml.example` to `config.yaml` and fill in provider keys and policy defaults.
- Set workspace directory where the runtime can read/write files and artifacts.

## Run
```bash
./gm run "Refactor this function"
```

## Optional
- Configure environment variables for API keys.
- Use systemd or a process supervisor to keep the service running.
