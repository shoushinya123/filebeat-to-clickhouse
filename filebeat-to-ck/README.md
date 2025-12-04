# Filebeat to ClickHouse 转换器

这是一个用 Golang 编写的转换器，接收 Filebeat 的输出（支持多种格式），并将其转换并写入 ClickHouse。

## 功能特性

- ✅ 支持 Elasticsearch bulk API 格式（Filebeat 默认输出格式）
- ✅ 支持 Logstash HTTP 输出
- ✅ 支持直接 JSON 事件数组
- ✅ 支持单个 JSON 事件
- ✅ 自动字段映射和转换
- ✅ 使用 ClickHouse HTTP 接口（最保险的方式）
- ✅ 单体实例，轻量级
- ✅ Docker 容器化部署
- ✅ 外置配置文件

## 快速开始

### 1. 构建 Docker 镜像

```bash
docker build -t filebeat-to-ck:latest .
```

### 2. 配置

编辑 `config.yaml`：

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

### 3. 运行

```bash
docker run -d \
  --name filebeat-to-ck \
  -p 8080:8080 \
  -v $(pwd)/config.yaml:/etc/filebeat-to-ck/config.yaml:ro \
  -e CONFIG_PATH=/etc/filebeat-to-ck/config.yaml \
  --network filebeat-network \
  filebeat-to-ck:latest
```

## API 接口

### 1. Elasticsearch Bulk API（推荐）

Filebeat 默认使用此格式：

```bash
POST http://filebeat-to-ck:8080/_bulk
Content-Type: application/x-ndjson

{"index":{"_index":"filebeat"}}
{"@timestamp":"2025-12-04T10:00:00Z","message":"test log"}
{"index":{"_index":"filebeat"}}
{"@timestamp":"2025-12-04T10:00:01Z","message":"another log"}
```

### 2. 事件数组

```bash
POST http://filebeat-to-ck:8080/events
Content-Type: application/json

[
  {
    "@timestamp": "2025-12-04T10:00:00Z",
    "message": "test log"
  }
]
```

### 3. 单个事件

```bash
POST http://filebeat-to-ck:8080/filebeat
Content-Type: application/json

{
  "@timestamp": "2025-12-04T10:00:00Z",
  "message": "test log"
}
```

### 4. 健康检查

```bash
GET http://filebeat-to-ck:8080/health
```

## 与 Filebeat 集成

在 `filebeat.yml` 中配置：

```yaml
output.elasticsearch:
  enabled: true
  hosts: ["http://filebeat-to-ck:8080"]
  index: "filebeat-%{+yyyy.MM.dd}"
  template.enabled: false
  ilm.enabled: false
```

## 数据转换

转换器会自动将 Filebeat 事件转换为 ClickHouse 表结构：

- `@timestamp` → `timestamp` (DateTime)
- `message` → `message` (String)
- `container.name` → `container` (String)
- `host.name` → `host_name` (String)
- `docker.container.*` → `docker_container_*` (String)
- `agent.*` → `agent_*` (String)
- 完整事件 → `raw_json` (String)

## 开发

### 本地运行

```bash
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

## 日志

日志级别通过 `config.yaml` 中的 `log_level` 配置：
- `debug`: 详细日志
- `info`: 一般信息（默认）
- `warn`: 警告
- `error`: 错误

## 故障排查

1. **检查服务状态**:
   ```bash
   curl http://localhost:8080/health
   ```

2. **查看日志**:
   ```bash
   docker logs filebeat-to-ck
   ```

3. **检查 ClickHouse 连接**:
   确保 ClickHouse 服务可访问，并且表已创建。

4. **验证数据写入**:
   ```bash
   docker exec clickhouse clickhouse-client --query "SELECT count() FROM logs.logs_table"
   ```
