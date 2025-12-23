# 阶段1: 构建前端
FROM node:18-alpine AS fe-builder
WORKDIR /app/fe
# 复制 package.json（先检查是否有 pnpm-lock.yaml）
COPY fe/package.json ./
# 如果有 pnpm-lock.yaml 就复制，没有就不复制
RUN npm install -g pnpm && pnpm install
COPY fe/ ./
RUN pnpm run build

# 阶段2: 构建后端
FROM golang:1.23.9-alpine AS server-builder
WORKDIR /app
COPY . .
# 从 fe-builder 复制前端构建产物
COPY --from=fe-builder /app/fe/dist ./server/static/fe
# 构建后端
RUN cd server && go mod tidy && go build -o ThinkForge-server main.go

# 阶段3: 最终镜像
FROM alpine:latest
WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata
ENV TZ=Asia/Shanghai

COPY --from=server-builder /app/server/ThinkForge-server /app/
COPY --from=server-builder /app/server/static/ /app/static/
COPY --from=server-builder /app/server/manifest/config/config_demo.yaml /app/manifest/config/config.yaml

EXPOSE 8000
CMD ["/app/ThinkForge-server"]