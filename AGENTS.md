# AGENTS.md

个人资产记录应用（Thingspan）：React 前端 + FastAPI 后端 + SQLite，单容器 Docker 部署。
设计文档见 `docs/plan.md`，用户部署说明见 `README.md`。仓库无自动化测试，改动验证靠手动冒烟（uvicorn + curl / 脚本）。

## 常用命令

```bash
# 后端（需要 uv）
cd backend
uv sync                                  # 改 pyproject.toml 后执行，同步 uv.lock
uv run uvicorn app.main:app --reload --port 8000
uv run alembic revision --autogenerate -m "desc"   # 改 models.py 后生成迁移
uv run alembic upgrade head                        # 应用启动时也会自动迁移

# 前端（Node 20+）
cd frontend
npm install                              # 改 package.json 后执行，同步 package-lock.json
npm run dev                              # http://localhost:5173，/api 代理到 127.0.0.1:8000
npm run build                            # tsc（noUnusedLocals 严格）+ vite，必须通过

# 本地联调：后端要托管前端页面，需手动同步构建产物
cp -r frontend/dist/* backend/app/static/

# Docker（uv.lock 与 package-lock.json 必须与依赖同步，镜像用 --frozen / npm ci）
# 打包上传流程见 docs/Docker镜像打包上传.md，一键执行 scripts/docker-push.sh
docker build -t pigzho/thingspan:latest .
```

## 关键约定

- **启动校验**：`.env` 的 `APP_PASSWORD` / `JWT_SECRET` 为空或占位值（`admin`/`change-me`）时启动直接报错退出（`main.py: validate_secrets`）。本地开发配置在 `backend/.env`（gitignored）。
- **时区**：全站以 `TZ`（默认 Asia/Shanghai）本地 naive 时间存储与展示；成本计算、提醒、调度器都基于该时区的"今天"，不要引入 UTC。
- **数据库**：SQLite 文件在 `$DATA_DIR/thingspan.db`。改模型必须生成 Alembic 迁移；SQLite 给已有表加 NOT NULL 列必须带 `server_default`（参考 `versions/a43a5729bc18`）。
- **前端产物**：`frontend/dist/` 与 `backend/app/static/` 均 gitignored；镜像由 Dockerfile 多阶段构建。`main.py` 的 SPA 回退路由只在 `app/static/index.html` 存在时注册，`/api/*` 未知路径 404。
- **成本口径**（`services/cost.py`）：in_use = 价格/已用天数；sold = (买−卖)/持有天数；broken = 价格/至损坏日天数；expired = 价格/有效天数。
- **类别勾选参数**（`models.py` Category）：`has_warranty` / `has_expiry` / `can_sell` / `can_break` / `has_serial` / `has_model` 六个布尔勾选，无自由自定义字段。状态校验（`api/assets.py`）：进入 sold 必填 sale_date + sale_price、进入 broken 必填 broken_date，且只在状态流转时校验 `can_sell`/`can_break` 勾选（存量售出/损坏资产在类别取消勾选后仍可编辑，不拦截）；离开 sold/broken 清空对应字段；已处于 sold/broken 时写入这些字段仍做必填校验（不允许置 null）。
- **保修推算**：类别勾选 `has_warranty` 且资产填写了 `warranty_months` 时，保修结束日 = 购买日 + 月数×30 天（月数在新建/编辑资产时填写，不同资产可不同，不在类别上配置）。新建或请求带 `purchase_date` / `warranty_months` / `category_id` 时强制重算（force，月数置空则清空结束日期），否则保留已有结束日期；`api/assets.py` 的 `_apply_warranty` 是唯一入口，前端 `AssetDetail.tsx` 只展示推算预览（不提交 `warranty_end_date`）——不要破坏这条链路。
- **类别勾选参数**（`models.py` Category）：`has_warranty` / `has_expiry` / `can_sell` / `can_break` / `has_serial` / `has_model` 六个布尔勾选，无自由自定义字段。状态校验（`api/assets.py`）：进入 sold 必填 sale_date + sale_price、进入 broken 必填 broken_date，且只在状态流转时校验 `can_sell`/`can_break` 勾选（存量售出/损坏资产在类别取消勾选后仍可编辑，不拦截）；离开 sold/broken 清空对应字段；已处于 sold/broken 时写入这些字段仍做必填校验（不允许置 null）。
- **提醒扫描**（`services/reminder.py`）：窗口匹配（基准日 ∈ [今天, 今天+lead]），每资产每天最多一封、每档最多一次；`ReminderLog.sent=False` 表示发送失败，次日自动重试；只有勾选 `has_expiry` 类别的资产到期才会自动标记 `expired`。
- **前端状态**：Token 存 localStorage；`lib/api.ts` 遇 401 自动用 refresh 换新后重试一次，失败跳登录页；后端 refresh 轮换并吊销旧 jti（内存实现，重启失效）。

## 变更检查清单

- 改 `models.py` → 生成并检查 Alembic 迁移（SQLite server_default 注意）
- 改前端 → `npm run build` 必须通过（tsc 严格模式会拒绝未使用导入）
- 改 pyproject/package.json → 同步锁文件，否则 Docker 构建失败
- 涉及状态/成本/提醒/动态字段 → 先读对应 services/api 文件确认现有约束
- 不提交：`.env`、`data/`、`frontend/dist/`、`backend/app/static/`
