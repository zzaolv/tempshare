# --- Stage 1: Build ---
# 使用官方的 Go 镜像作为构建环境
FROM golang:1.24-alpine AS builder

# 设置工作目录
WORKDIR /app

# 复制 go.mod 和 go.sum 文件并下载依赖
# 这一步是单独的，以便利用 Docker 的层缓存
COPY go.mod go.sum ./
RUN go mod download

# 复制所有源代码
COPY . .

# 构建应用。
# -o /app/tempshare 指定输出文件名为 tempshare
# CGO_ENABLED=0 禁用 CGO，构建静态链接的二进制文件
# -ldflags "-s -w" 去除调试信息，减小最终二进制文件的大小
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/tempshare -ldflags="-s -w" .

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