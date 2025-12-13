# 数据库多驱动支持 - 迁移指南

## 📋 实现概述

本项目现在支持PostgreSQL和SQLite3两种数据库，通过统一的抽象接口实现。

## 🏗️ 架构设计

```
┌─────────────────────────────────────────┐
│          Application Layer              │
│  (service, face, dao)                   │
└──────────────┬──────────────────────────┘
               │
┌──────────────▼──────────────────────────┐
│      Database Abstraction Layer        │
│  (database.DB, database.Tx)            │
└──────────────┬──────────────────────────┘
               │
    ┌──────────┴──────────┐
    │                        │
┌───▼────────┐      ┌────────▼──────┐
│ PostgreSQL │      │   SQLite3     │
│  Driver    │      │    Driver     │
└────────────┘      └────────────────┘
```

## 📦 核心组件

### 1. 数据库接口 (`database/driver.go`)

- `DB`: 数据库连接接口
- `Tx`: 事务接口
- `Row`, `Rows`, `Result`: 结果接口
- `Dialect`: SQL方言接口

### 2. 驱动实现

- **PostgreSQL** (`database/postgres.go`): 使用 `pgx/v5`
- **SQLite3** (`database/sqlite3.go`): 使用 `database/sql` + `go-sqlite3`

### 3. 辅助函数 (`database/builder.go`)

- `InsertWithID()`: 自动处理插入并获取ID的差异
- `GetPlaceholderFormat()`: 获取正确的占位符格式
- `BuildJSONColumn()`: 获取JSON列类型
- `BuildAutoIncrement()`: 获取自增列定义

## 🔄 迁移步骤

### 步骤1: 更新配置

**旧配置：**
```yaml
pg_urn: "postgres://user:pass@localhost:5432/db"
```

**新配置（推荐）：**
```yaml
db_driver: "postgres"  # 或 "sqlite3"
db_dsn: "postgres://user:pass@localhost:5432/db"
```

**向后兼容：** 旧的 `pg_urn` 配置仍然支持。

### 步骤2: 更新DAO层函数签名

**旧代码：**
```go
func UserInsert(p *protos.UserReq, tx *pgx.Tx) (uid int64, e error)
```

**新代码：**
```go
import "github.com/liuhengloveyou/passport/database"

func UserInsert(db database.DB, p *protos.UserReq, tx database.Tx) (uid int64, e error)
```

### 步骤3: 更新函数实现

**旧代码：**
```go
if tx != nil {
    err = (*tx).QueryRow(context.Background(), sql, vals...).Scan(&uid)
} else {
    err = common.DBPool.QueryRow(context.Background(), sql, vals...).Scan(&uid)
}
```

**新代码：**
```go
ctx := context.Background()
if tx != nil {
    err = tx.QueryRow(ctx, sql, vals...).Scan(&uid)
} else {
    err = db.QueryRow(ctx, sql, vals...).Scan(&uid)
}
```

### 步骤4: 使用辅助函数简化代码

**使用 `InsertWithID()` 自动处理差异：**

```go
// 旧代码：手动处理RETURNING
sql, vals, err := sq.Insert(table).SetMap(data).
    Suffix("RETURNING uid").
    PlaceholderFormat(sq.Dollar).
    ToSql()
// ... 执行查询

// 新代码：自动处理
uid, err := database.InsertWithID(ctx, db, tx, "users", data)
```

## 📝 完整迁移示例

### 示例：UserInsert函数

**迁移前：**
```go
package dao

import (
    "context"
    "github.com/jackc/pgx/v5"
    "github.com/liuhengloveyou/passport/common"
    sq "github.com/Masterminds/squirrel"
)

func UserInsert(p *protos.UserReq, tx *pgx.Tx) (uid int64, e error) {
    data := map[string]interface{}{
        "password": p.Password,
        "cellphone": p.Cellphone,
        // ...
    }
    
    sql, vals, err := sq.Insert("users").
        SetMap(data).
        Suffix("RETURNING uid").
        PlaceholderFormat(sq.Dollar).
        ToSql()
    
    if tx != nil {
        err = (*tx).QueryRow(context.Background(), sql, vals...).Scan(&uid)
    } else {
        err = common.DBPool.QueryRow(context.Background(), sql, vals...).Scan(&uid)
    }
    
    return uid, err
}
```

**迁移后：**
```go
package dao

import (
    "context"
    "github.com/liuhengloveyou/passport/common"
    "github.com/liuhengloveyou/passport/database"
    "github.com/liuhengloveyou/passport/protos"
)

func UserInsert(p *protos.UserReq, tx database.Tx) (uid int64, e error) {
    data := map[string]interface{}{
        "password": p.Password,
        "cellphone": p.Cellphone,
        // ...
    }
    
    ctx := context.Background()
    
    // 使用辅助函数自动处理PostgreSQL和SQLite3的差异
    uid, err := database.InsertWithID(ctx, common.DB, tx, "users", data)
    if err != nil {
        return -1, err
    }
    
    return uid, nil
}
```

## 🔍 数据库差异对照表

| 特性 | PostgreSQL | SQLite3 | 处理方式 |
|------|-----------|---------|---------|
| 占位符 | `$1, $2...` | `?` | `GetPlaceholderFormat()` |
| 获取插入ID | `RETURNING uid` | `last_insert_rowid()` | `InsertWithID()` |
| JSON类型 | `JSONB` | `TEXT` | `BuildJSONColumn()` |
| 自增列 | `BIGSERIAL` | `INTEGER PRIMARY KEY AUTOINCREMENT` | `BuildAutoIncrement()` |
| 并发写入 | ✅ 支持 | ⚠️ 有限支持 | SQLite3使用单连接 |

## ⚠️ 注意事项

1. **向后兼容性**：
   - `common.DBPool` 仍然可用（仅PostgreSQL）
   - 旧的 `pg_urn` 配置仍然支持
   - 可以逐步迁移，不需要一次性修改所有代码

2. **SQLite3限制**：
   - 建议用于开发、测试或小规模应用
   - 生产环境建议使用PostgreSQL
   - 某些PostgreSQL特性不可用（如JSONB操作符）

3. **性能考虑**：
   - PostgreSQL: 适合高并发、大规模应用
   - SQLite3: 适合低并发、小规模应用或嵌入式场景

4. **事务处理**：
   - 两种数据库都支持事务
   - SQLite3的事务需要小心处理并发

## 🧪 测试

### 使用SQLite3进行单元测试

```go
func TestUserInsert(t *testing.T) {
    // 使用内存数据库
    db, err := database.NewDB(database.DriverSQLite3, ":memory:")
    if err != nil {
        t.Fatal(err)
    }
    defer db.Close()
    
    // 初始化表结构
    // ...
    
    // 执行测试
    uid, err := UserInsert(db, &protos.UserReq{
        Cellphone: "13800138000",
        Password:  "123456",
    }, nil)
    
    assert.NoError(t, err)
    assert.Greater(t, uid, int64(0))
}
```

## 📚 相关文件

- `database/driver.go`: 核心接口定义
- `database/postgres.go`: PostgreSQL实现
- `database/sqlite3.go`: SQLite3实现
- `database/builder.go`: 辅助函数
- `database/README.md`: 使用文档
- `common/common.go`: 数据库初始化

## 🚀 下一步

1. 逐步迁移DAO层函数
2. 更新单元测试使用SQLite3内存数据库
3. 根据实际需求选择数据库类型
4. 在生产环境使用PostgreSQL

