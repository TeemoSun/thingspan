# ---- Stage 1: 前端构建 ----
FROM node:20-alpine AS frontend
WORKDIR /build
COPY frontend/package.json frontend/package-lock.json ./
RUN --mount=type=cache,target=/root/.npm npm ci
COPY frontend/ ./
RUN npm run build

# ---- Stage 2: 后端纯静态编译 ----
FROM golang:1.24-alpine AS backend-builder
WORKDIR /build
ENV GOPROXY=https://goproxy.cn,direct
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -buildvcs=false -trimpath -ldflags="-s -w" -o /build/thingspan ./cmd/server

# ---- Stage 3: 极简运行时 ----
FROM alpine:3.20 AS runtime
ENV DATA_DIR=/data \
    TZ=Asia/Shanghai

RUN apk add --no-cache tzdata ca-certificates && \
    addgroup -g 1000 -S appuser && \
    adduser -u 1000 -S appuser -G appuser && \
    mkdir -p /data /app/static && \
    chown -R appuser:appuser /data /app

WORKDIR /app

# 从构建阶段拷入静态二进制与前端静态产物
COPY --from=backend-builder --chown=appuser:appuser /build/thingspan /app/thingspan
COPY --from=frontend --chown=appuser:appuser /build/dist /app/static

USER appuser
EXPOSE 8000

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD ["/app/thingspan", "-healthcheck"]

CMD ["/app/thingspan"]
