# VMCat Server Mode - 多阶段构建
# 仅构建 serve 模式（无 GUI 依赖）

# --- 前端构建 ---
FROM node:20-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# --- Go 构建 ---
FROM golang:1.23-alpine AS builder
RUN apk add --no-cache gcc musl-dev sqlite-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/frontend/dist ./frontend/dist
# 构建仅 serve 模式的二进制（不依赖 Wails GUI）
RUN CGO_ENABLED=1 go build -tags headless -ldflags="-s -w" -o vmcat .

# --- 运行时 ---
FROM alpine:3.19
RUN apk add --no-cache ca-certificates sqlite-libs
WORKDIR /app
COPY --from=builder /app/vmcat .
EXPOSE 9600
ENV VMCAT_PORT=9600
ENTRYPOINT ["/app/vmcat", "serve"]
