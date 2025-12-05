# 构建阶段
FROM golang:1.21-alpine AS builder

WORKDIR /build

# 复制依赖文件
COPY go.mod go.sum* ./
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o filebeat-to-ck main.go

# 运行阶段
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /build/filebeat-to-ck .

# 创建配置目录
RUN mkdir -p /etc/filebeat-to-ck

# 暴露端口
EXPOSE 8080

# 运行应用
CMD ["./filebeat-to-ck"]





