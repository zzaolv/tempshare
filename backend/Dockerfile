# backend/Dockerfile

# --- Stage 1: Build ---
# 使用官方的 Go 镜像作为构建环境
FROM golang:1.24-alpine AS builder

# ✨ 修复点: 为 SQLite 安装 C 语言构建工具链 (CGO 依赖)
RUN apk add --no-cache build-base

# 设置工作目录
WORKDIR /app

# 1. 单独复制 go.mod 和 go.sum
COPY go.mod go.sum ./

# 2. 使用 Go modules 的缓存来下载依赖项
# 这比 `go mod download` 更高效，因为它只在 go.mod 变化时才重新下载
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# 3. 复制所有源代码到容器中
COPY . .

# 4. 构建应用，并利用 Go 的构建缓存
# -o /app/tempshare 指定输出文件名为 tempshare
# ✨ 修复点: 移除 CGO_ENABLED=0 来启用 CGO 编译, 这是 SQLite 驱动所必需的
# -ldflags "-s -w" 去除调试信息，减小最终二进制文件的大小
RUN --mount=type=cache,target=/root/.cache/go-build \
    GOOS=linux go build -o /app/tempshare -ldflags="-s -w" .

# --- Stage 2: Final Image ---
# 使用一个非常小的基础镜像 (alpine) 来构建最终镜像
FROM alpine:latest

# 设置工作目录
WORKDIR /app

# 从 builder 阶段复制构建好的二进制文件
COPY --from=builder /app/tempshare .

# Alpine 镜像可能需要 ca-certificates 来进行 HTTPS 调用 (例如连接 S3)
# 还需要 tzdata 来正确处理时区
RUN apk --no-cache add ca-certificates tzdata

# 暴露我们在配置中设置的端口 (默认 8080)
# 这只是元数据，实际端口映射在 docker-compose.yml 中完成
EXPOSE 8080

# 定义容器启动时执行的命令
# 容器将直接运行我们的 Go 应用
ENTRYPOINT ["/app/tempshare"]