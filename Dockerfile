# 构建后端
FROM golang:1.23.9-alpine AS server-builder
WORKDIR /app

# 安装构建依赖
RUN apk add --no-cache git ca-certificates tzdata

# 设置 Go 环境变量
ENV GO111MODULE=on \
    GOPROXY=https://goproxy.cn,direct \
    CGO_ENABLED=0

# 先复制 go.mod 和 go.sum，利用 Docker 缓存
COPY server/go.mod server/go.sum ./server/
RUN cd server && go mod download

# 复制整个项目代码
COPY . .

# 构建后端，确保依赖都被包含
RUN cd server && \
    go mod tidy && \
    go mod verify && \
    go build -v -o ThinkForge-server main.go

# 最终镜像
FROM alpine:latest
WORKDIR /app

# 安装运行时依赖
RUN apk --no-cache add ca-certificates tzdata

# 设置时区
ENV TZ=Asia/Shanghai

# 复制后端构建产物
COPY --from=server-builder /app/server/ThinkForge-server /app/
COPY --from=server-builder /app/server/static/ /app/static/
COPY --from=server-builder /app/server/manifest/config/config_demo.yaml /app/manifest/config/config.yaml

# 暴露端口
EXPOSE 8000

# 启动命令
CMD ["/app/ThinkForge-server"]