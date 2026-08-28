# Thingspan 个人资产记录

记录手机、电脑、会员等资产的购买信息与日均成本，在到期前 30 / 7 / 1 天通过邮件提醒。

- 单用户，密码与环境变量配置，JWT 双 Token 认证
- SQLite 数据库 + Alembic 自动迁移，无需额外数据库
- 类别支持模板（数码产品 / 会员 / 其他）与自定义字段
- 单容器 Docker 镜像，一条命令即可运行

## 快速开始（Docker）

```bash
cp .env.example .env
# 编辑 .env：修改 APP_PASSWORD、JWT_SECRET，配置 SMTP 与 MAIL_TO

# 方式一：docker compose（推荐，端口与数据卷已在 compose.yaml 定义）
docker compose up -d

# 方式二：docker run
docker run -d \
  --name thingspan \
  -p 19234:8000 \
  -v "$(pwd)/data:/data" \
  --env-file .env \
  pigzho/thingspan:latest
```

访问 `http://localhost:19234`，使用 `.env` 中的 `APP_PASSWORD` 登录。

- SQLite 数据库自动创建于挂载的 `data` 目录（`/data/thingspan.db`）
- 版本升级：拉取新镜像并重启容器，Alembic 自动升级数据库结构
- 修改 `APP_PASSWORD` 后重启容器即生效
- `APP_PASSWORD` 与 `JWT_SECRET` 为必填项且不接受默认占位值，未配置时容器启动会直接报错退出

## 环境变量

| 变量 | 说明 | 默认 |
|---|---|---|
| `APP_PASSWORD` | 登录密码 | `change-me` |
| `JWT_SECRET` | JWT 签名密钥，建议 `openssl rand -hex 32` 生成 | `change-me` |
| `ACCESS_TOKEN_EXPIRE_MINUTES` | Access Token 有效期 | `30` |
| `REFRESH_TOKEN_EXPIRE_DAYS` | Refresh Token 有效期 | `30` |
| `TZ` | 时区 | `Asia/Shanghai` |
| `DATA_DIR` | 数据目录（SQLite 存放处） | `/data` |
| `SMTP_HOST` / `SMTP_PORT` | SMTP 服务器（465 用 SSL，587 用 STARTTLS） | 空 |
| `SMTP_USER` / `SMTP_PASSWORD` | 发信账号与授权码 | 空 |
| `MAIL_TO` | 提醒接收邮箱 | 空 |
| `REMINDER_LEAD_DAYS` | 提前提醒天数，逗号分隔 | `30,7,1` |
| `REMINDER_CHECK_HOUR` | 每日检查发送时间（时区小时） | `9` |

> 未配置 SMTP 时应用照常运行，仅不发邮件。

## 使用说明

- **类别**：数码产品模板支持品牌/型号/序列号/保修结束日期（可配置保修月数自动推算），会员模板支持到期日期（到期自动标记已过期），均可追加自定义字段
- **资产**：新建默认「使用中」，可标记「已售出」（需售出日期与价格）或「已损坏」（需损坏日期）
- **日均成本**：使用中 = 价格 ÷ 已用天数；已售出 =（买入 − 卖出）÷ 持有天数；已损坏 = 价格 ÷ 使用天数；已过期 = 价格 ÷ 有效天数
- **提醒**：到期前 30 / 7 / 1 天各发一封邮件至 `MAIL_TO`，每档只发一次，可在「提醒记录」页忽略

## 本地开发

```bash
# 根目录配置环境变量
cp .env.example .env

# 后端（Python 3.11+，需要 uv）
cd backend
uv sync
uv run uvicorn app.main:app --reload --port 8000

# 前端（Node 20+）
cd frontend
npm install
npm run dev            # http://localhost:5173，/api 已代理到 8000
```
