**English** | [简体中文](./README.zh-CN.md)

# Passport

Passport is a Go user-center / identity service: accounts, login, sessions, multi-tenant RBAC, and (from **v4**) **one tenant with multiple organizations**.

Module path: [`github.com/liuhengloveyou/passport/v4`](https://github.com/liuhengloveyou/passport)

[![Go Reference](https://pkg.go.dev/badge/github.com/liuhengloveyou/passport/v4.svg)](https://pkg.go.dev/github.com/liuhengloveyou/passport/v4)
[![Release](https://img.shields.io/github/v/release/liuhengloveyou/passport)](https://github.com/liuhengloveyou/passport/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](./LICENSE)

## Features

- User register / login / logout / profile / password recovery
- Cookie or memory session store
- Multi-tenant SaaS model (one user belongs to one tenant)
- **Organizations under a tenant** (e.g. stores / sites) — v4
- RBAC with Casbin (domain = `tenant-{tenantId}-org-{orgId}`)
- Departments scoped by organization
- Data scope helpers: `all` / `dept` / `self`
- WeChat (MP / Mini Program) and Alipay H5 OAuth sessions
- SMS providers (e.g. Tencent Cloud)
- Run as a standalone HTTP service or embed into `net/http` / Gin

## v4: Multi-organization under one tenant

| Concept | Role |
| --- | --- |
| **Tenant** | Account boundary; a user still belongs to one tenant |
| **Organization** | Business unit under a tenant (e.g. a store) |
| **Department / RBAC / data scope** | Scoped to tenant + organization |

Request rules:

- APIs that need org context must send header **`X-Org-Id`**
- Casbin domain looks like: `tenant-{tenantId}-org-{orgId}`
- CORS allows `X-Org-Id`
- `-init` / seed ensures the root tenant has at least one organization and joins the root user

## Layout

| Package | Responsibility |
| --- | --- |
| `face/http` | HTTP entry, routing, auth / access filters |
| `face/core` | Session helpers, `ParseOrgID` / `SessionOrgID`, JSON body |
| `face/user` | User APIs |
| `face/tenant` | Tenant, members, departments, tenant tree |
| `face/access` | Roles, policies, permission dictionary |
| `face/admin` | Platform admin APIs |
| `face/sms` / `face/wx` / `face/ali` | SMS, WeChat, Alipay |
| `service/org.service.go` | Organization CRUD & membership |
| `service/datascope.go` | Resolve data scope per org |

## Quick start

### Standalone service

```bash
# copy and edit config (sample keys only — do not commit secrets)
# passport.conf.yaml is gitignored
go run ./cmd/passport -c passport.conf.yaml
```

Minimal config sketch:

```yaml
pid_file: "/tmp/passport.pid"
addr: ":8080"
log_dir: "./logs"
log_level: "debug"

db_driver: "postgres"
db_dsn: "host=localhost user=passport password=passport123 dbname=passport port=5432 sslmode=disable TimeZone=Asia/Shanghai"

session_store_type: "cookie" # or mem
session_expire: 0            # -1 delete; 0 session; >0 seconds

root_tenant_id: 10000

sms: ""
api_conf:
  "*":
    need_access: true
```

Passport owns auth for `/usercenter` APIs; your business handlers can stay separate.

### Embed in `net/http`

```go
package main

import (
	"net/http"
	"time"

	passport "github.com/liuhengloveyou/passport/v4/face/http"
	passportprotos "github.com/liuhengloveyou/passport/v4/protos"
)

func main() {
	options := &passportprotos.OptionStruct{
		LogDir:   "./logs",
		LogLevel: "debug",
		DBDriver: "postgres",
		DBDSN:    "host=localhost user=passport password=passport123 dbname=passport port=5432 sslmode=disable",
	}
	http.Handle("/usercenter", passport.InitAndRunHttpApi(options))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello passport"))
	})

	s := &http.Server{Addr: ":8080", ReadTimeout: 10 * time.Minute, WriteTimeout: 10 * time.Minute}
	_ = s.ListenAndServe()
}
```

### Embed in Gin

```go
engine.Any("/user", gin.WrapH(passport.InitAndRunHttpApi(options)))
```

## HTTP API convention

- Entry: `POST|GET /usercenter`
- Select API via header `X-API: user/login` (or path, depending on deployment)
- Org-scoped APIs: also send `X-Org-Id: <orgId>`
- Session: cookie (and optional business `session` header for H5 flows)

Example login:

```bash
curl -v -X POST -H "X-API: user/login" -H "USE-COOKIE: true" -d '{
  "cellphone": "15360651247",
  "password": "123456"
}' "http://127.0.0.1:10000/usercenter"
```

Example add role in an organization:

```bash
curl -v -X POST \
  -H "X-API: access/addRoleForUser" \
  -H "X-Org-Id: 10001" \
  --cookie "go-session-id=..." \
  -d '{"uid": 123, "value": "role1"}' \
  "http://127.0.0.1:10000/usercenter"
```

Full field definitions, every endpoint, SQL schema, and error codes: see **[README.zh-CN.md](./README.zh-CN.md)**.

## Response format

```json
{ "code": 0, "data": {} }
```

```json
{ "code": -1000, "msg": "error message" }
```

Common org-related codes: `-2008` org not found, `-2009` org required, `-2010` org name duplicate.

## Database

PostgreSQL (or SQLite3 for lighter setups). Startup can ensure schema. Key tables: `users`, `tenants`, `organizations`, `org_members`, `departments` (with `org_id`), `permission`, closure tables, Casbin rules.

See Chinese README for full DDL examples.

## Versioning

| Tag | Notes |
| --- | --- |
| **v4.x** | Multi-org under tenant; module path `/v4`; `X-Org-Id` required for org APIs |
| v3.x | Previous module path `/v3` |

Breaking when upgrading to v4: change imports to `/v4`, send `X-Org-Id`, and migrate RBAC domains to `tenant-{tid}-org-{oid}`.

## License

[MIT](./LICENSE)
