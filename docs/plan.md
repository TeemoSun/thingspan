# Thingspan 个人资产记录应用 — 设计方案

> 记录手机、电脑、会员等资产的购买信息与日均成本，提供到期提醒。
> 单用户自部署应用，一条 `docker run` 命令即可使用。

## 1. 目标与范围

- 记录资产的购买信息（价格、日期、品牌、型号、序列号等）与日均成本
- 按类别组织资产，支持自定义类别，类别参数（保修期/到期日期/售出/损坏/序列号/型号）通过勾选配置
- 支持资产状态流转：使用中 / 已售出 / 已损坏 / 已过期
- 基于保修结束日期 / 到期日期，在到期前 30 / 7 / 1 天发送邮件提醒
- 单容器 Docker 镜像发布到 DockerHub，数据落在挂载的 Data 目录

## 2. 技术栈

| 层 | 选型 |
|---|---|
| 前端 | React 18 + Vite + TypeScript，shadcn/ui（Tailwind），Tabler Icons，TanStack Query，React Router |
| 后端 | FastAPI + SQLAlchemy 2 + Pydantic v2，UV 管理依赖，APScheduler 定时任务 |
| 认证 | JWT 双 Token（Access 短期 + Refresh 长期轮换），密码启动时 bcrypt 哈希比对 |
| 数据库 | SQLite（文件位于 `$DATA_DIR`），Alembic 迁移，启动时自动 `alembic upgrade head` |
| 时区 | 环境变量配置，默认 `Asia/Shanghai`，影响提醒调度与成本计算 |
| 部署 | 多阶段 Docker 构建，单容器（FastAPI 托管前端静态文件），发布至 DockerHub |

## 3. 项目结构

```
thingspan/
├── backend/
│   ├── app/
│   │   ├── main.py            # 启动：迁移 → 建表 → 初始化 → 起调度器
│   │   ├── config.py          # pydantic-settings 读取 ENV
│   │   ├── database.py        # SQLite 连接与 Session
│   │   ├── models.py          # Category / Asset / ReminderLog
│   │   ├── schemas.py         # Pydantic 请求/响应模型
│   │   ├── security.py        # bcrypt 密码校验 + JWT 签发/校验
│   │   ├── api/
│   │   │   ├── auth.py        # 登录 / 刷新
│   │   │   ├── categories.py  # 类别 CRUD + 字段配置
│   │   │   ├── assets.py      # 资产 CRUD + 筛选 + 状态切换 + 成本
│   │   │   ├── dashboard.py   # 汇总统计
│   │   │   └── reminders.py   # 提醒记录 / 忽略
│   │   ├── services/
│   │   │   ├── cost.py        # 日均成本计算
│   │   │   ├── reminder.py    # 到期扫描与去重
│   │   │   └── email.py       # SMTP 邮件发送
│   │   └── scheduler.py       # APScheduler 每日任务
│   ├── alembic/               # 迁移脚本
│   ├── pyproject.toml         # uv 依赖管理
│   └── Dockerfile
├── frontend/
│   ├── src/
│   │   ├── components/        # shadcn/ui 组件与业务组件
│   │   ├── pages/
│   │   │   ├── Login.tsx
│   │   │   ├── Dashboard.tsx
│   │   │   ├── Assets.tsx        # 资产列表
│   │   │   ├── AssetDetail.tsx   # 详情 / 编辑 / 状态切换
│   │   │   ├── Categories.tsx    # 类别与字段配置
│   │   │   └── Reminders.tsx     # 提醒记录
│   │   ├── lib/               # API 客户端、JWT 刷新、日期工具
│   │   └── main.tsx
│   ├── package.json
│   └── Dockerfile
├── Dockerfile               # 根目录多阶段 Dockerfile（前端构建 + 后端运行）
├── .env.example
└── README.md                # 部署说明
```

## 4. 环境变量（.env.example）

```
# 认证
APP_PASSWORD=                # 登录密码，可随意修改，重启生效
JWT_SECRET=                  # 固定签名密钥，建议固定避免重启后旧 Refresh 失效
ACCESS_TOKEN_EXPIRE_MINUTES=30
REFRESH_TOKEN_EXPIRE_DAYS=30

# 时区与数据
TZ=Asia/Shanghai             # 默认上海
DATA_DIR=/data               # SQLite 数据库与日志目录

# 邮件提醒（SMTP）
SMTP_HOST=smtp.qq.com
SMTP_PORT=465
SMTP_USER=your@qq.com
SMTP_PASSWORD=               # 邮箱授权码
MAIL_TO=your@qq.com          # 提醒接收邮箱

# 提醒策略
REMINDER_LEAD_DAYS=30,7,1    # 提前天数，可改
REMINDER_CHECK_HOUR=9        # 每日检查发送时间
```

## 5. 数据模型

### Category（类别）

| 字段 | 说明 |
|---|---|
| id | 主键 |
| name | 类别名称 |
| has_warranty | 勾选「保修期」：资产有保修结束日期，配置保修月数后按购买日自动推算 |
| has_expiry | 勾选「到期日期」：资产有到期日期，到期后自动标记已过期 |
| can_sell | 勾选「可售出」：资产可标记已售出（填写售出日期/价格） |
| can_break | 勾选「可损坏」：资产可标记已损坏（填写损坏日期） |
| has_serial | 勾选「序列号」：资产表单出现序列号字段 |
| has_model | 勾选「型号」：资产表单出现型号字段 |
| warranty_months | 保修月数（勾选保修期时可配置） |
| created_at | 创建时间 |

类别参数只能通过勾选配置（不可自定义字段），新建该类资产时表单出现对应字段、状态操作受勾选约束。

### Asset（资产）

| 字段 | 说明 |
|---|---|
| id | 主键 |
| category_id | 所属类别 |
| name | 名称 |
| brand / model / serial_number | 品牌（通用）/ 型号 / 序列号（型号、序列号按类别勾选显示） |
| purchase_date | 购买日期（必填） |
| purchase_price | 购买价格（必填） |
| warranty_end_date | 保修结束日期（可空，类别勾选「保修期」时显示） |
| expiry_date | 到期日期（可空，类别勾选「到期日期」时显示） |
| status | `in_use` / `sold` / `broken` / `expired`，默认 `in_use` |
| sale_date / sale_price | 售出日期 / 售出价格（已售出时必填） |
| broken_date | 损坏日期（已损坏时必填） |
| notes | 备注 |
| created_at / updated_at | 时间戳 |

细节：

- 保修期按「购买日 + 保修月数」自动推算结束日期，可手动修改（类别勾选保修期并配置月数）
- 未勾选「可售出」的类别禁止标记已售出（状态流转时校验）；未勾选「可损坏」同理；已售出/已损坏资产在类别取消勾选后仍可正常编辑
- 售出 / 损坏操作要求填写对应日期，售出还需售出价格
- 到期日过期的「使用中」资产（类别勾选「到期日期」）由每日任务自动标记为 `expired`

### ReminderLog（提醒记录，去重）

| 字段 | 说明 |
|---|---|
| id | 主键 |
| asset_id | 资产 |
| target_date | 提醒基准日期（保修结束或到期日） |
| lead_days | 提前天数档位（30 / 7 / 1） |
| sent_at | 最后尝试时间 |
| sent | 是否发送成功（False 表示失败待重试） |
| dismissed | 是否手动忽略 |

唯一约束：`(asset_id, target_date, lead_days)`，保证每档最多发送一次。

## 6. 认证（JWT 双 Token）

- 启动时校验 `APP_PASSWORD` / `JWT_SECRET`：为空或占位值（`admin` / `change-me`）直接报错退出
- 启动时对 `APP_PASSWORD` 做 bcrypt 哈希，仅存哈希内存中
- `POST /api/auth/login`：校验密码 → 返回 Access（30 分钟）+ Refresh（30 天）
- `POST /api/auth/refresh`：校验 Refresh 并轮换签发新 Refresh（旧 jti 吊销，进程内存实现）
- 前端请求带 Bearer Access，过期后自动用 Refresh 续期；Refresh 过期则跳回登录页

## 7. 日均成本计算

| 状态 | 公式 |
|---|---|
| 使用中 | 购买价 ÷ 自购买起天数 |
| 已售出 | (购买价 − 售出价) ÷ (售出日 − 购买日) |
| 已损坏 | 购买价 ÷ (损坏日 − 购买日) |
| 已过期 | 购买价 ÷ (到期日 − 购买日) |

- 购买当天天数为 0 时按 0 计
- 仪表盘汇总「当前日均总成本」= 各资产成本之和
- 资产详情展示单条资产的成本明细

## 8. 到期提醒

- APScheduler 每日 `REMINDER_CHECK_HOUR` 按 `TZ` 换算后执行，启动时补跑一次
- 窗口匹配：扫描 `in_use` 资产，基准日期（保修结束 / 到期，取非空者）∈ [今天, 今天+lead] 即命中该档
- 每资产每天最多发送一封（按升序命中最小档位），每档最多一次（`ReminderLog` 唯一约束去重）
- `ReminderLog.sent=False` 表示发送失败，次日扫描自动重试，成功后置 True
- 只有 membership 模板资产到期自动标记 `expired`（数码产品保修结束仅停止提醒，不改变状态）
- 提醒记录页可查看全部发送记录（含失败待重试）并手动「忽略」

## 9. API 设计

```
POST   /api/auth/login          # 登录，返回双 Token
POST   /api/auth/refresh        # 刷新 Token

GET    /api/categories          # 类别列表（含字段定义）
POST   /api/categories          # 新建类别
PUT    /api/categories/{id}     # 更新类别（含字段配置）
DELETE /api/categories/{id}     # 删除类别（有资产时拒绝）

GET    /api/assets              # 列表，筛选：类别 / 状态 / 搜索
POST   /api/assets              # 新建资产（默认 in_use）
GET    /api/assets/{id}         # 详情（含成本计算）
PUT    /api/assets/{id}         # 编辑 / 切换状态（售出/损坏填附加字段）
DELETE /api/assets/{id}         # 删除

GET    /api/dashboard           # 总资产数、总投入、日均总成本、30 天内到期列表
GET    /api/reminders           # 提醒记录列表
POST   /api/reminders/{id}/dismiss   # 忽略某条提醒
```

## 10. 前端页面

| 页面 | 功能 |
|---|---|
| 登录页 | 密码登录，Token 自动续期 |
| 仪表盘 | 资产总数、总投入、当前日均总成本、30 天内到期资产 |
| 资产列表 | 卡片/表格展示，按类别、状态筛选，关键字搜索 |
| 资产详情 | 按类别勾选的参数渲染表单（型号/序列号/保修/到期）；状态切换（售出 → 填日期与售价；损坏 → 填日期）；成本展示 |
| 类别管理 | 创建/编辑类别，勾选类别拥有的参数 |
| 提醒记录 | 查看发送历史，手动忽略 |

界面语言：中文。

## 11. Docker 打包与部署

### 构建（多阶段，`Dockerfile` 位于仓库根目录）

1. Stage 1 `node:20-alpine`：`npm ci` + 构建前端 → `dist` 静态文件
2. Stage 2 `python:3.12-slim`：uv（锁定版本 `0.11.28`）安装后端依赖，拷贝应用与前端 `dist`，FastAPI 托管静态文件
3. 启动命令（exec 形式，直接运行 uvicorn）：迁移在应用 lifespan 内以代码方式执行（`alembic command.upgrade`）
4. 声明 `VOLUME ["/data"]`、`HEALTHCHECK` 探测 `/api/health`；依赖锁文件必须同步（`--frozen` / `npm ci`）

### 使用

```bash
cp .env.example .env   # 修改 APP_PASSWORD、JWT_SECRET、SMTP、DATA_DIR 等
docker run -d \
  -p 19234:8000 \
  -v /path/to/data:/data \
  --env-file .env \
  pigzho/thingspan:latest
```

- SQLite 数据库文件自动创建于 `$DATA_DIR` 下
- `APP_PASSWORD` / `JWT_SECRET` 为空或占位值时容器启动报错退出
- 版本升级时拉取新镜像重启即可，Alembic 自动迁移数据库

## 12. 实施状态

已全部完成（2026-08）：项目骨架、后端模型与迁移、bcrypt + JWT 双 Token、类别/资产 CRUD 与动态字段、成本计算与仪表盘、提醒调度 + SMTP 邮件（含失败重试）、前端 6 页面、Docker 镜像（已本地构建验证）。
待用户操作：`docker push pigzho/thingspan:latest` 发布 DockerHub。
