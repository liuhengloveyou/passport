# 数据库抽象层使用指南

本包提供了统一的数据库抽象接口，支持PostgreSQL和SQLite3两种数据库。

## 快速开始

### 1. 配置数据库

在配置文件中设置数据库类型和连接字符串：

```yaml
# 方式1：使用新的配置（推荐）
db_driver: "postgres"  # 或 "sqlite3"
db_dsn: "postgres://user:password@localhost:5432/dbname?sslmode=disable"

# 方式2：向后兼容旧配置（仅PostgreSQL）
pg_urn: "postgres://user:password@localhost:5432/dbname?sslmode=disable"
```

### 2. 在代码中使用

#### 基本使用

```go
import (
    "github.com/liuhengloveyou/passport/common"
    "github.com/liuhengloveyou/passport/database"
)

// 数据库已通过common包自动初始化
// 使用 common.DB 访问数据库

// 查询单行
ctx := context.Background()
var uid int64
var nickname string
err := common.DB.QueryRow(ctx, "SELECT uid, nickname FROM users WHERE uid = $1", 10001).Scan(&uid, &nickname)

// 查询多行
rows, err := common.DB.Query(ctx, "SELECT uid, nickname FROM users WHERE tenant_id = $1", 10002)
defer rows.Close()
for rows.Next() {
    var uid int64
    var nickname string
    rows.Scan(&uid, &nickname)
    // 处理数据
}

// 执行更新
result, err := common.DB.Exec(ctx, "UPDATE users SET nickname = $1 WHERE uid = $2", "新昵称", 10001)
affected, _ := result.RowsAffected()
```

#### 事务使用

```go
ctx := context.Background()
tx, err := common.DB.Begin(ctx)
if err != nil {
    return err
}
defer tx.Rollback(ctx)

// 在事务中执行操作
_, err = tx.Exec(ctx, "INSERT INTO users (nickname, password) VALUES ($1, $2)", "user1", "pass1")
if err != nil {
    return err
}

// 提交事务
if err = tx.Commit(ctx); err != nil {
    return err
}
```

#### 使用辅助函数插入并获取ID

```go
import "github.com/liuhengloveyou/passport/database"

data := map[string]interface{}{
    "nickname": "testuser",
    "password": "123456",
    "tenant_id": 10002,
}

// 自动处理PostgreSQL的RETURNING和SQLite3的last_insert_rowid()
uid, err := database.InsertWithID(ctx, common.DB, nil, "users", data)
```

## 数据库差异处理

### SQL占位符

- **PostgreSQL**: 使用 `$1, $2, $3...`
- **SQLite3**: 使用 `?`

代码中可以使用 `database.GetPlaceholderFormat()` 获取正确的占位符格式。

### 获取插入的ID

- **PostgreSQL**: 使用 `RETURNING uid` 子句
- **SQLite3**: 使用 `last_insert_rowid()`

推荐使用 `database.InsertWithID()` 函数，它会自动处理这些差异。

### JSON类型

- **PostgreSQL**: `JSONB`
- **SQLite3**: `TEXT`

使用 `database.BuildJSONColumn()` 获取正确的类型定义。

### 自增列

- **PostgreSQL**: `BIGSERIAL`
- **SQLite3**: `INTEGER PRIMARY KEY AUTOINCREMENT`

使用 `database.BuildAutoIncrement()` 获取正确的定义。

## 迁移现有代码

### 从 pgx.Tx 迁移

**旧代码：**
```go
func UserInsert(p *protos.UserReq, tx *pgx.Tx) (uid int64, e error) {
    // ...
    if tx != nil {
        err = (*tx).QueryRow(context.Background(), sql, vals...).Scan(&uid)
    } else {
        err = common.DBPool.QueryRow(context.Background(), sql, vals...).Scan(&uid)
    }
}
```

**新代码：**
```go
func UserInsert(p *protos.UserReq, tx database.Tx) (uid int64, e error) {
    // ...
    ctx := context.Background()
    if tx != nil {
        err = tx.QueryRow(ctx, sql, vals...).Scan(&uid)
    } else {
        err = common.DB.QueryRow(ctx, sql, vals...).Scan(&uid)
    }
}
```

### 使用InsertWithID简化代码

**旧代码：**
```go
sql, vals, err := sq.Insert(table).SetMap(data).Suffix("RETURNING uid").PlaceholderFormat(sq.Dollar).ToSql()
if tx != nil {
    err = (*tx).QueryRow(context.Background(), sql, vals...).Scan(&uid)
} else {
    err = common.DBPool.QueryRow(context.Background(), sql, vals...).Scan(&uid)
}
```

**新代码：**
```go
uid, err := database.InsertWithID(ctx, common.DB, tx, "users", data)
```

## 配置示例

### PostgreSQL配置

```yaml
db_driver: "postgres"
db_dsn: "postgres://user:password@localhost:5432/passport?sslmode=disable&TimeZone=Asia/Shanghai"
```

### SQLite3配置

```yaml
db_driver: "sqlite3"
db_dsn: "./data/passport.db"
# 或使用内存数据库（仅用于测试）
# db_dsn: ":memory:"
```

## 注意事项

1. **SQLite3限制**：
   - 建议使用单连接（已自动配置）
   - 不支持并发写入
   - 某些PostgreSQL特性不可用

2. **向后兼容**：
   - 旧的 `common.DBPool` 仍然可用（仅PostgreSQL）
   - 旧的 `pg_urn` 配置仍然支持
   - 建议逐步迁移到新的 `common.DB` 接口

3. **性能**：
   - PostgreSQL适合生产环境
   - SQLite3适合开发、测试或小规模应用

## 测试

```go
// 测试时可以使用SQLite3内存数据库
db, err := database.NewDB(database.DriverSQLite3, ":memory:")
if err != nil {
    t.Fatal(err)
}
defer db.Close()

// 执行测试...
```

