# ---- Stage 1: 前端构建 ----
FROM node:20-alpine AS frontend
WORKDIR /build
COPY frontend/package.json frontend/package-lock.json ./
RUN --mount=type=cache,target=/root/.npm npm ci
COPY frontend/ ./
RUN npm run build

# ---- Stage 2: 后端依赖构建 ----
FROM python:3.12-alpine AS backend-builder
COPY --from=ghcr.io/astral-sh/uv:0.11.28 /uv /usr/local/bin/uv
ENV UV_COMPILE_BYTECODE=1 \
    UV_LINK_MODE=copy \
    UV_PROJECT_ENVIRONMENT=/opt/venv
WORKDIR /build
COPY backend/pyproject.toml backend/uv.lock ./
RUN --mount=type=cache,target=/root/.cache/uv \
    uv sync --frozen --no-dev --no-install-project

# ---- Stage 3: 最终运行镜像 ----
FROM python:3.12-alpine AS runtime
ENV PYTHONUNBUFFERED=1 \
    PATH="/opt/venv/bin:$PATH" \
    DATA_DIR=/data \
    TZ=Asia/Shanghai

RUN apk add --no-cache tzdata
WORKDIR /app

# 从构建阶段拷入独立的虚拟环境（不含 uv 工具）
COPY --from=backend-builder /opt/venv /opt/venv

# 拷入 Alembic 迁移配置与脚本
COPY backend/alembic ./alembic
COPY backend/alembic.ini ./alembic.ini

# 拷入后端运行代码
COPY backend/app ./app

# 拷入前端静态产物
COPY --from=frontend /build/dist ./app/static

VOLUME ["/data"]
EXPOSE 8000

HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD python -c "import urllib.request; urllib.request.urlopen('http://127.0.0.1:8000/api/health', timeout=3)"

CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8000"]

