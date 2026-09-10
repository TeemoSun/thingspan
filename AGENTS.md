# AGENTS.md

个人资产记录应用（Thingspan）：React 前端 + Go (Chi + modernc.org/sqlite) 后端 + SQLite，单容器 Docker 部署。
原 Python 后端已归档于 `backend_py/`（gitignored）。用户部署说明见 `README.md`。

## 常用命令

```bash
# 后端（Go >= 1.23，CGO_ENABLED=0 纯静态）
cd backend
go test -v ./...                         # 运行单元测试与集成测试
go run ./cmd/server                      # 启动后端服务（端口 8000）
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o thingspan ./cmd/server # 构建静态二进制
./thingspan -healthcheck                 # 运行本地原生健康检查探针

# 前端（Node 20+）
cd frontend
npm install                              # 改 package.json 后执行，同步 package-lock.json
npm run dev                              # http://localhost:5173，/api 代理到 127.0.0.1:8000
npm run build                            # tsc（noUnusedLocals 严格）+ vite，必须通过

# 本地联调：后端要托管前端页面，可将构建产物拷入静态目录
cp -r frontend/dist/* backend/static/

# Docker 构建
docker build -t ghcr.io/teemosun/thingspan:latest .
```

## 关键约定

- **启动校验**：`.env` 的 `APP_PASSWORD` / `JWT_SECRET` 为空或占位值（`admin`/`change-me`）时启动直接报错退出（`config.ValidateSecrets`）。本地开发配置在项目根目录 `.env`（gitignored）。
- **纯 Go SQLite**：选用 `modernc.org/sqlite`，纯 Go 静态编译无 gcc/glibc 依赖。设置 WAL 模式、外键约束、busy_timeout=5000ms 与最大连接数 1，彻底避免竞争锁库。
- **数据库迁移**：物理文件位于 `$DATA_DIR/thingspan.db`。由 `internal/database/migrations.go` 自动幂等建表并检查/平滑升级存量字段，兼容旧版本 Python Alembic 历史库。
- **时区**：全站以 `TZ`（默认 Asia/Shanghai）本地时间存储与展示；成本计算、提醒、调度器都基于该时区的"今天"，禁止使用 UTC 导致跨日偏差。
- **成本口径**（`internal/services/cost.go`）：
  - `in_use` = 价格 / 已用天数（若勾选 `has_expiry` 且有到期日，按 到期日 − 购买日 计算，与 expired 一致，不随今日增长）；
  - `sold` = (买入价 − 售出价) / 持有天数；
  - `broken` = 价格 / 至损坏日天数；
  - `expired` = 价格 / 有效天数。
- **类别勾选参数**（`models.Category`）：`has_warranty` / `has_expiry` / `can_sell` / `can_break` / `has_serial` / `has_model` 六个布尔勾选，无自由自定义字段。
  - 状态流转校验：进入 `sold` 必填 `sale_date` + `sale_price` 且类别必须勾选 `can_sell`；进入 `broken` 必填 `broken_date` 且类别必须勾选 `can_break`；离开 `sold`/`broken` 时自动清空对应字段；存量售出/损坏资产再次编辑时禁止置 null。
- **保修推算**：类别勾选 `has_warranty` 且资产填写了 `warranty_months` 时，保修结束日 = 购买日 + 月数 × 30 天。新建或请求包含 `purchase_date` / `warranty_months` / `category_id` 时强制重算，前端 `AssetDetail.tsx` 只展示推算预览（不提交 `warranty_end_date`）——不要破坏这条链路。
- **到期状态自动同步**（`internal/services/cost.go: SyncExpiryStatus`）：勾选 `has_expiry` 类别的资产，读取（列表/详情）与写入（新建/编辑）时按今日自动判定——到期日已过 → `expired`，到期日改到未来或清空 → 恢复 `in_use`。
- **提醒扫描与 SMTP**（`internal/services/reminder.go`）：后台每日 9:00（可配）及启动时自动扫描，到期前 30 / 7 / 1 天发送格式化邮件；每资产每天最多一封、每档最多一次；发送失败记录 `sent=false`，次日自动重试。
- **容器健康检查**：二进制内置 `-healthcheck` 探针（调用 `/healthz` 校验 SQLite 读写后返回退出码 0/1），容器移除对外部 wget/python 命令依赖。

## 变更检查清单

- 改数据结构 → 在 `internal/database/migrations.go` 维护幂等迁移与兼容检查
- 改后端代码 → `cd backend && go test -v ./...` 必须全部通过
- 改前端 → `cd frontend && npm run build` 必须通过（tsc 严格模式）
- 不提交：`.env`、`data/`、`frontend/dist/`、`backend/static/`、`backend/thingspan`、`backend_py/`
