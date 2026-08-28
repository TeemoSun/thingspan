# Docker 镜像打包与发布流程

本项目镜像托管于 **GitHub Container Registry (`ghcr.io`)**，支持 **GitHub Actions 自动构建发布** 与 **本地一键脚本构建**。

## 自动化构建（推荐）

仓库已配置 GitHub Actions 工作流（`.github/workflows/docker-publish.yml`）：

- **触发条件**：当代码推送到 `main` 分支或发布版本标签（`v*.*.*`）时，GitHub Actions 会自动触发多阶段构建，并将镜像推送至 `ghcr.io/teemosun/thingspan`。
- **自动标签**：
  - `latest`：始终指向 `main` 分支最新构建。
  - `YYYYMMDD`：按构建发布日期归档（如 `20260828`）。
  - `sha-xxxxxxx`：基于 Git Commit SHA。
  - `vX.Y.Z`：基于 Git Release 标签。

---

## 部署机使用镜像

默认端口 `19234`（compose 已定义端口与数据卷）：

```bash
cp .env.example .env   # 修改 APP_PASSWORD、JWT_SECRET、SMTP 等配置

# 方式一：docker compose（推荐）
docker compose up -d

# 方式二：docker run
docker run -d \
  --name thingspan \
  -p 19234:8000 \
  -v "$(pwd)/data:/data" \
  --env-file .env \
  ghcr.io/teemosun/thingspan:latest
```

> **提示**：公开镜像无需执行 `docker login`，任何机器均可直接拉取。

---

## 本地手动构建与推送（可选）

如需在本地构建并推送到 GHCR：

### 1. 登录 GitHub Container Registry

```bash
# 使用具备 packages:write 权限的 GitHub Personal Access Token (PAT) 登录
echo "$GITHUB_TOKEN" | docker login ghcr.io -u <your-github-username> --password-stdin
```

### 2. 执行一键构建推送脚本

仓库已提供 `scripts/docker-push.sh`（需 `chmod +x`）：

```bash
bash scripts/docker-push.sh
```

或手动构建：

```bash
docker build -t ghcr.io/teemosun/thingspan:latest -t ghcr.io/teemosun/thingspan:$(date +%Y%m%d) .
docker push ghcr.io/teemosun/thingspan:latest
docker push ghcr.io/teemosun/thingspan:$(date +%Y%m%d)
```
