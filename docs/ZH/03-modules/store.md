# 模块设计: Store (存储层)

> 支持多后端的可扩展存储层设计

---

## 1. 设计目标

| 目标 | 说明 |
| :--- | :--- |
| **多后端** | 支持 FS / SQLite / PostgreSQL |
| **可扩展** | 新增后端只需实现接口 |
| **零切换成本** | 配置变更即切换，无代码改动 |
| **事务支持** | 可选的事务语义 |

---

## 2. 架构设计

```mermaid
graph TB
    subgraph Application["应用层"]
        Runtime[Runtime]
        Agent[Agent]
    end
    
    subgraph Store["存储抽象层"]
        Interface[Store Interface]
        Factory[Store Factory]
    end
    
    subgraph Backends["后端实现"]
        FS[FS Store<br/>JSONL + JSON]
        SQLite[SQLite Store]
        PG[PostgreSQL Store]
        Future[... 未来扩展]
    end
    
    Runtime --> Interface
    Agent --> Interface
    Interface --> Factory
    Factory --> FS
    Factory --> SQLite
    Factory --> PG
    Factory --> Future
    
    style Interface fill:#9cf
    style Factory fill:#f9c
```

---

## 3. 核心接口设计

### 3.1 Store 主接口

```go
package store

import (
    "context"
    "time"
)

// 数据类型定义见 data-model.md

// Store 是存储层的主接口
// 所有后端必须实现此接口
type Store interface {
    // 生命周期
    Open(ctx context.Context) error
    Close() error
    
    // 事件存储 (append-only)
    EventStore
    
    // 状态快照
    StateStore
    
    // 产物存储
    ArtifactStore
    
    // 注意: Transactional 是可选的，不在 Store 接口中
}

// EventStore 事件日志接口
type EventStore interface {
    // 追加事件 (幂等: 相同 ID 忽略)
    AppendEvent(ctx context.Context, event Event) error
    
    // 批量追加
    AppendEvents(ctx context.Context, events []Event) error
    
    // 查询事件
    GetEvent(ctx context.Context, id string) (Event, error)
    
    // 获取指定事件之后的所有事件
    GetEventsSince(ctx context.Context, afterEventID string) ([]Event, error)
    
    // 迭代所有事件 (用于重放)
    IterEvents(ctx context.Context, fn func(Event) error) error
}

// StateStore 状态快照接口
type StateStore interface {
    // 保存状态快照
    SaveState(ctx context.Context, state *State) error
    
    // 加载指定版本状态
    LoadState(ctx context.Context, version int64) (*State, error)
    
    // 加载最新状态
    LoadLatestState(ctx context.Context) (*State, error)
    
    // 保存检查点
    SaveCheckpoint(ctx context.Context, cp *Checkpoint) error
    
    // 加载最新检查点
    LoadLatestCheckpoint(ctx context.Context) (*Checkpoint, error)
    
    // 加载指定检查点
    LoadCheckpoint(ctx context.Context, id string) (*Checkpoint, error)
}

// ArtifactStore 产物存储接口
type ArtifactStore interface {
    // 保存产物
    SaveArtifact(ctx context.Context, artifact *Artifact) error
    
    // 获取产物
    GetArtifact(ctx context.Context, id string) (*Artifact, error)
    
    // 列出产物
    ListArtifacts(ctx context.Context, filter ArtifactFilter) ([]Artifact, error)
    
    // 删除产物
    DeleteArtifact(ctx context.Context, id string) error
}

// Transactional 事务接口 (可选实现)
// 使用类型断言检查: if tx, ok := store.(Transactional); ok { ... }
type Transactional interface {
    // 开启事务
    Begin(ctx context.Context) (Tx, error)
}

// Tx 事务接口
type Tx interface {
    Store
    Commit() error
    Rollback() error
}

// WithTransaction 辅助函数: 如果 store 支持事务则使用，否则直接执行
func WithTransaction(ctx context.Context, store Store, fn func(Store) error) error {
    if tx, ok := store.(Transactional); ok {
        t, err := tx.Begin(ctx)
        if err != nil {
            return err
        }
        if err := fn(t); err != nil {
            t.Rollback()
            return err
        }
        return t.Commit()
    }
    // 不支持事务，直接执行
    return fn(store)
}
```

### 3.2 存储记录格式

```go
// 数据结构以 data-model.md 为准，这里仅展示存储记录格式
type EventRecord struct {
    ID        string          `json:"id"`
    Type      string          `json:"type"`
    Timestamp time.Time       `json:"timestamp"`
    Actor     string          `json:"actor,omitempty"`
    Subject   string          `json:"subject,omitempty"`
    Payload   json.RawMessage `json:"payload"`
}

func EncodeEvent(event Event) (EventRecord, []byte, error) {
    payload, err := json.Marshal(event)
    if err != nil {
        return EventRecord{}, nil, err
    }
    record := EventRecord{
        ID:        event.EventID(),
        Type:      event.EventType(),
        Timestamp: event.EventTimestamp(),
        Actor:     event.EventActor(),
        Subject:   event.EventSubject(),
        Payload:   payload,
    }
    return record, payload, nil
}
```

---

## 4. 工厂模式

### 4.1 Store Factory

```go
package store

import (
    "fmt"
    
    "gm-agent/pkg/store/fs"
    "gm-agent/pkg/store/sqlite"
    "gm-agent/pkg/store/postgres"
)

// Config 存储配置
type Config struct {
    // 后端类型: "fs" | "sqlite" | "postgres"
    Backend string `json:"backend" yaml:"backend"`
    
    // FS 配置
    FS *FSConfig `json:"fs,omitempty" yaml:"fs,omitempty"`
    
    // SQLite 配置
    SQLite *SQLiteConfig `json:"sqlite,omitempty" yaml:"sqlite,omitempty"`
    
    // PostgreSQL 配置
    Postgres *PostgresConfig `json:"postgres,omitempty" yaml:"postgres,omitempty"`
}

type FSConfig struct {
    BasePath string `json:"base_path" yaml:"base_path"`
}

type SQLiteConfig struct {
    Path string `json:"path" yaml:"path"`
}

type PostgresConfig struct {
    DSN string `json:"dsn" yaml:"dsn"`
}

// New 创建 Store 实例
func New(cfg Config) (Store, error) {
    switch cfg.Backend {
    case "fs", "":
        if cfg.FS == nil {
            return nil, fmt.Errorf("fs config is required")
        }
        return fs.New(cfg.FS.BasePath)
        
    case "sqlite":
        if cfg.SQLite == nil {
            return nil, fmt.Errorf("sqlite config is required")
        }
        return sqlite.New(cfg.SQLite.Path)
        
    case "postgres":
        if cfg.Postgres == nil {
            return nil, fmt.Errorf("postgres config is required")
        }
        return postgres.New(cfg.Postgres.DSN)
        
    default:
        return nil, fmt.Errorf("unknown backend: %s", cfg.Backend)
    }
}
```

### 4.2 配置示例

```yaml
# config.yaml

# 开发环境: 文件存储
store:
  backend: fs
  fs:
    base_path: ~/.gm-agent/data

---
# 测试环境: SQLite
store:
  backend: sqlite
  sqlite:
    path: ~/.gm-agent/data.db

---
# 生产环境: PostgreSQL
store:
  backend: postgres
  postgres:
    dsn: postgres://user:pass@localhost:5432/gm_agent?sslmode=disable
```

---

## 5. 后端实现

### 5.1 FS Store (文件系统)

```
~/.gm-agent/data/
├── workspaces/
│   └── ws_{id}/
│       ├── sessions/
│       │   └── s_{id}/
│       │       ├── state.json        # 状态快照
│       │       ├── events.jsonl      # 事件日志 (append-only)
│       │       ├── checkpoints.jsonl # 检查点列表
│       │       └── artifacts/        # 产物目录
│       │           └── {id}.json
│       └── metadata.json
└── global/
    └── config.json
```

```go
package fs

type FSStore struct {
    basePath string
    mu       sync.RWMutex
}

func (s *FSStore) AppendEvent(ctx context.Context, event Event) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    path := s.eventsPath()
    f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return err
    }
    defer f.Close()
    
    line, _ := json.Marshal(event)
    _, err = f.WriteString(string(line) + "\n")
    return err
}

// 原子写入：先写临时文件，再 rename
func (s *FSStore) SaveState(ctx context.Context, state *State) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    path := s.statePath()
    tmpPath := path + ".tmp"
    
    // 1. 写入临时文件
    data, err := json.MarshalIndent(state, "", "  ")
    if err != nil {
        return err
    }
    
    if err := os.WriteFile(tmpPath, data, 0644); err != nil {
        return err
    }
    
    // 2. Sync 确保数据落盘
    f, err := os.Open(tmpPath)
    if err != nil {
        return err
    }
    if err := f.Sync(); err != nil {
        f.Close()
        return err
    }
    f.Close()
    
    // 3. 原子 rename
    return os.Rename(tmpPath, path)
}
```

> [!CAUTION]
> **FS Store 一致性警告**
>
> FS Store **不保证崩溃一致性**：
> - Event 写入和 Checkpoint 写入是**两个独立操作**
> - 如果在两者之间崩溃，恢复时可能出现状态不一致
> - **生产环境必须使用 SQLite 或 PostgreSQL**
>
> FS Store 仅适用于：
> - 本地开发
> - 单次运行的脚本
> - 对数据丢失不敏感的场景
```

**优点**: 无依赖、Debug 友好、可 Git 管理  
**缺点**: 无事务、并发写入需外部锁

### 5.2 SQLite Store

```sql
-- schema.sql

CREATE TABLE IF NOT EXISTS events (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    timestamp DATETIME NOT NULL,
    actor TEXT,
    subject TEXT,
    data JSON NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_events_timestamp ON events(timestamp);
CREATE INDEX idx_events_subject ON events(subject);

CREATE TABLE IF NOT EXISTS states (
    version INTEGER PRIMARY KEY,
    data JSON NOT NULL,
    updated_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS checkpoints (
    id TEXT PRIMARY KEY,
    state_version INTEGER NOT NULL,
    last_event_id TEXT NOT NULL,
    timestamp DATETIME NOT NULL,
    FOREIGN KEY (state_version) REFERENCES states(version)
);

CREATE TABLE IF NOT EXISTS artifacts (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    name TEXT NOT NULL,
    path TEXT,
    content BLOB,
    metadata JSON,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

```go
package sqlite

type SQLiteStore struct {
    db *sql.DB
}

func (s *SQLiteStore) AppendEvent(ctx context.Context, event Event) error {
    record, payload, err := EncodeEvent(event)
    if err != nil {
        return err
    }
    _, err = s.db.ExecContext(ctx, `
        INSERT OR IGNORE INTO events (id, type, timestamp, actor, subject, data)
        VALUES (?, ?, ?, ?, ?, ?)
    `, record.ID, record.Type, record.Timestamp, record.Actor, record.Subject, payload)
    return err
}

// 事务支持
func (s *SQLiteStore) Begin(ctx context.Context) (Tx, error) {
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return nil, err
    }
    return &SQLiteTx{tx: tx, store: s}, nil
}
```

**优点**: 单文件、事务支持、索引查询  
**缺点**: 单进程写入

### 5.3 PostgreSQL Store

```go
package postgres

import (
    "github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
    pool *pgxpool.Pool
}

func New(dsn string) (*PostgresStore, error) {
    pool, err := pgxpool.New(context.Background(), dsn)
    if err != nil {
        return nil, err
    }
    return &PostgresStore{pool: pool}, nil
}

func (s *PostgresStore) AppendEvent(ctx context.Context, event Event) error {
    record, payload, err := EncodeEvent(event)
    if err != nil {
        return err
    }
    _, err = s.pool.Exec(ctx, `
        INSERT INTO events (id, type, timestamp, actor, subject, data)
        VALUES ($1, $2, $3, $4, $5, $6)
        ON CONFLICT (id) DO NOTHING
    `, record.ID, record.Type, record.Timestamp, record.Actor, record.Subject, payload)
    return err
}
```

**优点**: 多进程/分布式、完整事务、高并发  
**缺点**: 需要运维 PostgreSQL

---

## 6. 迁移策略

### 6.1 导出/导入

```go
// 导出数据
func Export(src Store) (*ExportData, error) {
    data := &ExportData{}
    
    // 导出状态
    state, _ := src.LoadLatestState(ctx)
    data.State = state
    
    // 导出事件
    src.IterEvents(ctx, func(e Event) error {
        data.Events = append(data.Events, e)
        return nil
    })
    
    // 导出产物
    artifacts, _ := src.ListArtifacts(ctx, ArtifactFilter{})
    data.Artifacts = artifacts
    
    return data, nil
}

// 导入数据
func Import(dst Store, data *ExportData) error {
    for _, e := range data.Events {
        dst.AppendEvent(ctx, e)
    }
    dst.SaveState(ctx, data.State)
    for _, a := range data.Artifacts {
        dst.SaveArtifact(ctx, &a)
    }
    return nil
}
```

### 6.2 CLI 命令

```bash
# 从文件迁移到 SQLite
gm store migrate --from fs --to sqlite

# 从 SQLite 迁移到 PostgreSQL  
gm store migrate --from sqlite --to postgres
```

---

## 7. 扩展新后端

实现新后端只需 3 步：

```go
// 1. 创建新包
package mystore

// 2. 实现 Store 接口
type MyStore struct {}

func (s *MyStore) Open(ctx context.Context) error { ... }
func (s *MyStore) Close() error { ... }
func (s *MyStore) AppendEvent(ctx context.Context, event Event) error { ... }
// ... 实现所有接口方法

// 3. 注册到 Factory
// store/factory.go
case "mystore":
    return mystore.New(cfg.MyStore)
```

---

## 8. 目录结构

```
pkg/store/
├── store.go          # 接口定义
├── factory.go        # 工厂函数
├── types.go          # 数据结构
├── fs/               # FS 实现
│   ├── fs.go
│   └── fs_test.go
├── sqlite/           # SQLite 实现
│   ├── sqlite.go
│   ├── schema.sql
│   └── sqlite_test.go
├── postgres/         # PostgreSQL 实现
│   ├── postgres.go
│   ├── schema.sql
│   └── postgres_test.go
└── testing/          # 测试工具
    └── mock.go
```

---

## 9. 总结

| 特性 | FS | SQLite | PostgreSQL |
| :--- | :---: | :---: | :---: |
| 无依赖 | ✅ | ✅ | ❌ |
| 事务 | ❌ | ✅ | ✅ |
| 并发写入 | ❌ | 单进程 | ✅ |
| 分布式 | ❌ | ❌ | ✅ |
| Debug 友好 | ✅ | ⚠️ | ❌ |
| 推荐场景 | MVP/开发 | 单机生产 | 企业级 |

通过接口抽象，**应用代码无需关心底层存储**，只需切换配置即可。
