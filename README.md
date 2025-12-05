# Filebeat to ClickHouse 转换器

一个用 Golang 编写的轻量级转换器，接收 Filebeat 的输出（支持多种格式），并将其转换并写入 ClickHouse。

## ✨ 特性

- ✅ 支持 Filebeat 的 Elasticsearch 输出格式（推荐）
- ✅ 支持 Filebeat 的 Logstash HTTP 输出
- ✅ 支持直接 JSON 接收
- ✅ 自动字段映射和转换
- ✅ 使用 ClickHouse HTTP 接口
- ✅ 单体实例，轻量级
- ✅ Docker 容器化部署
- ✅ 外置配置文件

## 🚀 快速开始

### 1. 克隆仓库

```bash
git clone https://github.com/your-username/filebeat-to-clickhouse.git
cd filebeat-to-clickhouse
```

### 2. 构建 Docker 镜像

```bash
cd filebeat-to-ck
docker build -t filebeat-to-ck:latest .
cd ..
```

### 3. 配置

编辑 `filebeat-to-ck/config.yaml`：

```yaml
server:
  host: "0.0.0.0"
  port: 8080

clickhouse:
  host: "clickhouse"
  port: 8123
  database: "logs"
  table: "logs_table"
  user: "default"
  password: ""

log_level: "info"
```

### 4. 启动服务

```bash
docker-compose up -d
```

### 5. 初始化 ClickHouse 表

```bash
docker exec -i clickhouse clickhouse-client < clickhouse-init/init.sql
```

## 📋 项目结构

```
.
├── filebeat-to-ck/          # 转换器项目
│   ├── main.go              # 主程序
│   ├── go.mod               # Go 模块定义
│   ├── Dockerfile           # Docker 构建文件
│   ├── config.yaml          # 配置文件
│   └── README.md            # 项目说明
├── docker-compose.yml       # Docker Compose 配置
├── filebeat.yml             # Filebeat 配置示例
├── clickhouse-init/         # ClickHouse 初始化脚本
│   └── init.sql
└── README.md               # 本文件
```

## 🔧 配置 Filebeat

在 `filebeat.yml` 中配置输出到转换器：

```yaml
output.elasticsearch:
  enabled: true
  hosts: ["http://filebeat-to-ck:8080"]
  index: "filebeat-%{+yyyy.MM.dd}"
  template.enabled: false
  ilm.enabled: false
```

## 📊 支持的输入格式

### 1. Elasticsearch Bulk API（推荐）

```bash
POST http://filebeat-to-ck:8080/_bulk
Content-Type: application/x-ndjson

{"index":{}}
{"@timestamp":"2025-12-04T10:00:00Z","message":"test log"}
```

### 2. Logstash HTTP

```bash
POST http://filebeat-to-ck:8080/logstash
Content-Type: application/json

{"@timestamp":"2025-12-04T10:00:00Z","message":"test log"}
```

### 3. 直接 JSON

```bash
POST http://filebeat-to-ck:8080/events
Content-Type: application/json

[{"@timestamp":"2025-12-04T10:00:00Z","message":"test log"}]
```

## 🔍 验证

### 检查服务状态

```bash
curl http://localhost:8080/health
```

### 查看数据

```bash
docker exec clickhouse clickhouse-client --query "SELECT count() FROM logs.logs_table"
docker exec clickhouse clickhouse-client --query "SELECT * FROM logs.logs_table ORDER BY timestamp DESC LIMIT 10"
```

## 📝 字段映射

转换器自动将 Filebeat 事件字段映射到 ClickHouse 表：

| Filebeat 字段 | ClickHouse 字段 | 类型 |
|--------------|----------------|------|
| @timestamp | timestamp | DateTime |
| message | message | String |
| container.name | container | String |
| host.name | host_name | String |
| docker.container.id | docker_container_id | String |
| docker.container.name | docker_container_name | String |
| agent.name | agent_name | String |
| agent.version | agent_version | String |
| log.file.path | log_file_path | String |
| (完整事件) | raw_json | String |

## 🛠️ 开发

### 本地运行

```bash
cd filebeat-to-ck
go mod download
go run main.go
```

### 测试

```bash
# 测试健康检查
curl http://localhost:8080/health

# 测试事件接收
curl -X POST http://localhost:8080/filebeat \
  -H "Content-Type: application/json" \
  -d '{"@timestamp":"2025-12-04T10:00:00Z","message":"test"}'
```

## 📄 许可证

MIT License

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📧 联系方式

如有问题，请提交 Issue。
