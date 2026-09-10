# Thingspan 个人资产记录

记录手机、电脑、会员等资产的购买信息与日均成本，在到期前 30 / 7 / 1 天通过邮件提醒。

- 单用户，密码与环境变量配置，JWT 双 Token 认证
- 纯 Go (Go >= 1.23 + Chi) 后端 + 纯 Go SQLite 驱动（`modernc.org/sqlite`，`CGO_ENABLED=0` 静态编译）
- 容器常驻物理内存仅约 **10MB**，镜像体积仅约 **25MB**
- 内置原生 `-healthcheck` 探针，无缝健康检查
- 自动幂等建表与针对老版本存量 SQLite 数据库的无痛平滑迁移
- 资产 6 项布尔参数自定义配置（保修期/到期日期/售出/损坏/序列号/型号）
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
  ghcr.io/teemosun/thingspan:latest
```

访问 `http://localhost:19234`，使用 `.env` 中的 `APP_PASSWORD` 登录。

- SQLite 数据库自动创建于挂载的 `data` 目录（`/data/thingspan.db`）
- 版本升级：拉取新镜像并重启容器，程序自动幂等同步与升级数据库结构
- 修改 `APP_PASSWORD` 后重启容器即生效
- `APP_PASSWORD` 与 `JWT_SECRET` 为必填项且不接受默认占位值，未配置时容器启动会直接报错退出

## 环境变量

| 变量 | 说明 | 默认 |
|---|---|---|
| `APP_PASSWORD` | 登录密码（必填，禁止默认占位值） | `change-me` |
| `JWT_SECRET` | JWT 签名密钥，建议 `openssl rand -hex 32` 生成 | `change-me` |
| `ACCESS_TOKEN_EXPIRE_MINUTES` | Access Token 有效期（分钟） | `30` |
| `REFRESH_TOKEN_EXPIRE_DAYS` | Refresh Token 有效期（天） | `30` |
| `TZ` | 时区 | `Asia/Shanghai` |
| `DATA_DIR` | 数据目录（SQLite 存放处） | `/data` |
| `PORT` | 监听端口 | `8000` |
| `STATIC_DIR` | 前端静态文件目录 | `./static` |
| `SMTP_HOST` / `SMTP_PORT` | SMTP 服务器（465 用 SSL，587 用 STARTTLS） | 空 |
| `SMTP_USER` / `SMTP_PASSWORD` | 发信账号与授权码 | 空 |
| `MAIL_TO` | 提醒接收邮箱 | 空 |
| `REMINDER_LEAD_DAYS` | 提前提醒天数，逗号分隔 | `30,7,1` |
| `REMINDER_CHECK_HOUR` | 每日检查发送时间（时区小时） | `9` |

> 未配置 SMTP 时应用照常运行，仅不发邮件。

## 使用说明

- **类别**：灵活勾选保修期（资产填保修月数自动推算结束日）、到期日期（到期自动标记已过期）、可售出、可损坏、序列号、型号等参数。
- **资产**：新建默认「使用中」，可标记「已售出」（需售出日期与价格）或「已损坏」（需损坏日期）。
- **日均成本**：使用中 = 价格 ÷ 已用天数（若勾选到期日按有效天数计算）；已售出 =（买入 − 卖出）÷ 持有天数；已损坏 = 价格 ÷ 至损坏日天数；已过期 = 价格 ÷ 有效天数。
- **提醒**：到期前 30 / 7 / 1 天各发一封格式化邮件至 `MAIL_TO`，每档只发一次，可在「提醒记录」页忽略。

## 本地开发

```bash
# 根目录配置环境变量
cp .env.example .env

# 后端（Go >= 1.23）
cd backend
go test -v ./...       # 运行测试
go run ./cmd/server    # 启动后端（8000 端口）

# 前端（Node 20+）
cd frontend
npm install
npm run dev            # http://localhost:5173，/api 已代理到 8000
npm run build          # 静态构建
```
