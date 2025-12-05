# Filebeat to ClickHouse Converter

一个基于 Golang 的转换器，用于接收 Filebeat 输出并写入 ClickHouse。支持多种 Filebeat 输出格式，使用 ClickHouse 最可靠的 JSONEachRow 接口进行数据写入。

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org/)
[![Filebeat](https://img.shields.io/badge/Filebeat-7.x%2B-orange.svg)](https://www.elastic.co/beats/filebeat)
[![ClickHouse](https://img.shields.io/badge/ClickHouse-20.x%2B-green.svg)](https://clickhouse.com/)

## 📋 目录

- [功能特性](#功能特性)
- [架构说明](#架构说明)
- [支持的版本](#支持的版本)
- [快速开始](#快速开始)
- [配置说明](#配置说明)
- [数据流转](#数据流转)
- [性能评估](#性能评估)
- [常见问题](#常见问题)
- [开发指南](#开发指南)

## ✨ 功能特性

- ✅ **多格式支持**：兼容 Elasticsearch Bulk API、Logstash HTTP、直接 JSON 等多种 Filebeat 输出格式
- ✅ **可靠写入**：使用 ClickHouse JSONEachRow 格式，确保数据可靠性
- ✅ **批量处理**：支持批量写入，提高性能
- ✅ **配置驱动**：所有配置外部化，无需修改代码
- ✅ **Docker 支持**：提供完整的 Docker 和 Docker Compose 配置
- ✅ **无状态设计**：易于扩展和部署
- ✅ **实时传输**：低延迟，实时数据流转
- ✅ **灵活扩展**：框架支持 Kafka、Redis、TCP 等输入源（待实现）

## 🏗️ 架构说明

### 整体架构

```
┌─────────────┐         ┌──────────────┐         ┌─────────────┐
│  Filebeat   │ ──────> │ 转换器       │ ──────> │ ClickHouse  │
│  (日志收集) │ HTTP    │ (Golang)     │ HTTP    │ (数据存储)  │
└─────────────┘         └──────────────┘         └─────────────┘
```

### 数据流转过程

1. **Filebeat** 收集 Docker 容器日志
2. **Filebeat** 通过 HTTP 发送到转换器（使用 Elasticsearch Bulk API 格式）
3. **转换器** 接收、解析、转换数据格式
4. **转换器** 通过 HTTP 写入 ClickHouse（使用 JSONEachRow 格式）

### 技术实现

- **协议兼容**：转换器兼容 Elasticsearch Bulk API，无需修改 Filebeat 配置
- **格式转换**：将 Filebeat 事件格式转换为 ClickHouse 表结构
- **可靠写入**：使用 ClickHouse 的 JSONEachRow 接口，确保数据可靠性
- **配置驱动**：所有配置外部化，便于部署和维护

## 📦 支持的版本

### Filebeat

| 版本 | 支持状态 | 说明 |
|------|---------|------|
| 7.x | ✅ 完全支持 | 推荐使用 |
| 8.x | ✅ 完全支持 | **推荐版本** |
| **已测试** | Filebeat 8.11.0 | 生产验证 |

**原因**：
- 使用 `output.elasticsearch`（7.x 起支持）
- Elasticsearch Bulk API 格式稳定
- 协议向后兼容

### ClickHouse

| 版本 | 支持状态 | 说明 |
|------|---------|------|
| 20.x+ | ✅ 完全支持 | 基础支持 |
| 21.x+ | ✅ 完全支持 | 稳定版本 |
| 22.x+ | ✅ 完全支持 | 稳定版本 |
| 23.x+ | ✅ 完全支持 | 稳定版本 |
| 24.x+ | ✅ 完全支持 | **推荐版本** |
| 25.x+ | ✅ 完全支持 | **推荐版本** |
| **已测试** | ClickHouse 25.11.2.24 | 生产验证 |

**注意**：
- ClickHouse 25.x 需要密码认证（可通过环境变量配置）
- 使用 HTTP JSONEachRow 接口（20.x+ 支持）
- HTTP Basic Auth 认证

### Go 语言

| 版本 | 支持状态 | 说明 |
|------|---------|------|
| 1.19+ | ✅ 支持 | 基础支持 |
| 1.20+ | ✅ 支持 | 稳定版本 |
| 1.21+ | ✅ 支持 | **推荐版本**（当前使用） |
| 1.22+ | ✅ 支持 | 最新版本 |

### Docker

- **Docker**: 20.10+
- **Docker Compose**: v2.0+（推荐）

### 依赖库

- **Gin 框架**: v1.8.x - v1.10.x（当前 v1.9.1）
- **YAML 解析**: gopkg.in/yaml.v3 v3.0.x（当前 v3.0.1）

### 推荐版本组合

**生产环境**：
```
Filebeat: 8.11.0+
ClickHouse: 24.x 或 25.x
Go: 1.21+
Docker: 20.10+
```

**开发测试**：
```
Filebeat: 8.11.0
ClickHouse: latest (25.x)
Go: 1.21
Docker: 最新稳定版
```

## 🚀 快速开始

### 前置要求

- Docker 20.10+
- Docker Compose v2.0+
- Go 1.21+（如需本地开发）

### 方式一：使用 Docker Compose（推荐）

1. **克隆仓库**
```bash
git clone https://github.com/shoushinya123/filebeat-to-clickhouse.git
cd filebeat-to-clickhouse
```

2. **配置 ClickHouse 密码**

编辑 `docker-compose.yml`，设置环境变量：
```yaml
environment:
  CLICKHOUSE_PASSWORD: "your_password"
```

或使用环境变量：
```bash
export CLICKHOUSE_PASSWORD=your_password
```

同时更新 `filebeat-to-ck/config.yaml`：
```yaml
clickhouse:
  password: "your_password"
```

3. **启动服务**
```bash
docker-compose up -d
```

4. **验证服务**
```bash
# 检查转换器健康状态
curl http://localhost:8080/health

# 检查 ClickHouse
docker exec clickhouse clickhouse-client --password your_password --query "SELECT 1"

# 查看服务状态
docker-compose ps
```

5. **查看日志**
```bash
# 转换器日志
docker logs -f filebeat-to-ck

# Filebeat 日志
docker logs -f filebeat

# ClickHouse 日志
docker logs -f clickhouse
```

### 方式二：本地开发

1. **构建转换器**
```bash
cd filebeat-to-ck
go mod download
go build -o filebeat-to-ck main.go
```

2. **配置环境**
```bash
export CONFIG_PATH=./config.yaml
```

3. **运行转换器**
```bash
./filebeat-to-ck
```

4. **启动 ClickHouse**
```bash
docker run -d --name clickhouse \
  -p 8123:8123 -p 9000:9000 \
  -e CLICKHOUSE_PASSWORD=your_password \
  clickhouse/clickhouse-server:latest
```

5. **初始化数据库**
```bash
docker exec -i clickhouse clickhouse-client --password your_password < clickhouse-init/init.sql
```

## ⚙️ 配置说明

### 转换器配置 (`filebeat-to-ck/config.yaml`)

```yaml
server:
  host: "0.0.0.0"
  port: 8080

clickhouse:
  host: "clickhouse"  # Docker 网络中使用服务名，本地使用 localhost
  port: 8123
  database: "logs"
  table: "logs_table"
  user: "default"
  password: "your_password"  # 必须设置

log_level: "info"

# 支持的输入源配置（未来扩展）
inputs:
  elasticsearch:
    enabled: true
  logstash:
    enabled: false
  kafka:
    enabled: false
  redis:
    enabled: false
```

### Filebeat 配置 (`filebeat.yml`)

```yaml
filebeat.inputs:
  - type: container
    paths:
      - '/var/lib/docker/containers/*/*.log'
    json.keys_under_root: false
    json.add_error_key: true
    json.message_key: message
    processors:
      - add_docker_metadata:
          host: "unix:///var/run/docker.sock"

processors:
  - decode_json_fields:
      fields: ["message"]
      target: ""
      overwrite_keys: true
  - add_host_metadata:
      when.not.contains.tags: forwarded
  - add_docker_metadata: ~

# 输出配置 - 输出到转换器（兼容 Elasticsearch API）
output.elasticsearch:
  enabled: true
  hosts: ["http://filebeat-to-ck:8080"]  # 指向转换器
  index: "filebeat-%{+yyyy.MM.dd}"
  template.enabled: false
  ilm.enabled: false

logging.level: info
```

### ClickHouse 表结构 (`clickhouse-init/init.sql`)

```sql
-- 创建数据库
CREATE DATABASE IF NOT EXISTS logs;

-- 创建表用于接收 Filebeat 日志
CREATE TABLE IF NOT EXISTS logs.logs_table
(
    `timestamp` DateTime DEFAULT now(),
    `message` String DEFAULT '',
    `container` String DEFAULT '',
    `host_name` String DEFAULT '',
    `docker_container_id` String DEFAULT '',
    `docker_container_name` String DEFAULT '',
    `agent_name` String DEFAULT '',
    `agent_version` String DEFAULT '',
    `log_file_path` String DEFAULT '',
    `raw_json` String DEFAULT ''  -- 存储完整的 JSON 字符串
)
ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (timestamp)
SETTINGS index_granularity = 8192;
```

## 🔄 数据流转

### 详细流程

1. **Filebeat 收集日志**
   - 监控 Docker 容器日志文件：`/var/lib/docker/containers/*/*.log`
   - 解析 JSON 格式日志
   - 添加元数据（host, container, docker 等）

2. **Filebeat 发送数据**
   - 使用 `output.elasticsearch` 配置
   - 按照 **Elasticsearch Bulk API 格式**发送
   - HTTP POST 到转换器：`http://filebeat-to-ck:8080/_bulk`
   - 格式：每两行一组（action + document）

3. **转换器接收数据**
   - 接收 HTTP POST 请求
   - 解析 Bulk API 格式（按行分割）
   - 提取 document 行（实际数据）
   - 转换为 `FilebeatEvent` 结构

4. **数据格式转换**
   - 将 Filebeat 事件格式转换为 ClickHouse 表结构
   - 提取时间戳、消息、容器、主机等字段
   - 处理多种时间格式（RFC3339、ISO8601 等）

5. **写入 ClickHouse**
   - 使用 **JSONEachRow 格式**批量写入
   - HTTP POST 请求，带 Basic Auth 认证
   - 一次请求可写入多条记录

### 数据格式示例

**Filebeat 发送格式（Bulk API）**：
```
POST http://filebeat-to-ck:8080/_bulk
Content-Type: application/x-ndjson

{"index":{}}
{"@timestamp":"2025-12-04T10:00:00Z","message":"应用日志","container":{"name":"app"},"host":{"name":"server1"}}
{"index":{}}
{"@timestamp":"2025-12-04T10:01:00Z","message":"应用日志2","container":{"name":"app"},"host":{"name":"server1"}}
```

**转换器处理后的格式**：
```json
{
  "timestamp": "2025-12-04 10:00:00",
  "message": "应用日志",
  "container": "app",
  "host_name": "server1"
}
```

**ClickHouse 存储格式（JSONEachRow）**：
```
POST http://clickhouse:8123/?query=INSERT+INTO+logs.logs_table+FORMAT+JSONEachRow
Content-Type: application/x-ndjson

{"timestamp":"2025-12-04 10:00:00","message":"应用日志","container":"app","host_name":"server1"}
{"timestamp":"2025-12-04 10:01:00","message":"应用日志2","container":"app","host_name":"server1"}
```

### 关键要点

- **Filebeat → 转换器**：使用 ES 协议格式，但**不是**真正的 ES
- **转换器**：兼容 ES 协议，直接接收 HTTP 请求
- **转换器 → ClickHouse**：使用 JSONEachRow 格式写入

## 📊 性能评估

### 关键指标

- **吞吐量**：每秒处理的事件数（EPS - Events Per Second）
- **延迟**：从 Filebeat 发送到 ClickHouse 写入的时间（P99 延迟）
- **资源使用**：CPU、内存占用
- **错误率**：写入失败率
- **批量效率**：批量写入的性能提升

### 测试方法

#### 1. 压力测试

使用 Apache Bench 测试转换器：
```bash
# 准备测试数据
cat > test_bulk.json << EOF
{"index":{}}
{"@timestamp":"2025-12-04T10:00:00Z","message":"test message 1","host":{"name":"test"}}
{"index":{}}
{"@timestamp":"2025-12-04T10:00:01Z","message":"test message 2","host":{"name":"test"}}
EOF

# 执行压力测试
ab -n 10000 -c 100 -p test_bulk.json -T application/x-ndjson \
   http://localhost:8080/_bulk
```

#### 2. 监控指标

```bash
# 查看转换器日志
docker logs -f filebeat-to-ck

# 查看 ClickHouse 数据量
docker exec clickhouse clickhouse-client --password your_password \
  --query "SELECT count() FROM logs.logs_table"

# 查看 ClickHouse 写入性能
docker exec clickhouse clickhouse-client --password your_password \
  --query "SELECT count(), min(timestamp), max(timestamp) FROM logs.logs_table"
```

#### 3. 性能监控

```bash
# 监控转换器资源使用
docker stats filebeat-to-ck

# 监控 ClickHouse 资源使用
docker stats clickhouse
```

### 性能调优建议

1. **批量写入大小**
   - 调整 Filebeat 的批量大小配置
   - 转换器支持批量处理，一次写入多条记录

2. **ClickHouse 优化**
   - 优化表结构（分区、排序键）
   - 调整 `index_granularity` 参数
   - 使用合适的压缩算法

3. **扩展性**
   - 增加转换器实例（负载均衡）
   - 使用消息队列（Kafka）缓冲
   - ClickHouse 集群部署

4. **网络优化**
   - 使用 Docker 内部网络（减少延迟）
   - 调整 HTTP 超时设置
   - 启用 HTTP Keep-Alive

## ❓ 常见问题

### Q: 为什么使用 Elasticsearch 输出而不是直接输出到 ClickHouse？

**A**: Filebeat 原生不支持直接输出到 ClickHouse。使用 `output.elasticsearch` 配置可以让 Filebeat 按照 Elasticsearch Bulk API 格式发送数据，转换器兼容此格式，无需修改 Filebeat 配置。这样既利用了 Filebeat 的原生功能，又实现了到 ClickHouse 的转换。

### Q: 转换器支持哪些 Filebeat 输出格式？

**A**: 当前已实现支持：
- ✅ **Elasticsearch Bulk API**（主要使用）
- ✅ **Logstash HTTP**
- ✅ **直接 JSON**（单个事件或数组）

未来计划支持：
- 🔄 Logstash TCP（Lumberjack/Beats protocol）
- 🔄 Kafka
- 🔄 Redis（LIST/PUBSUB）
- 🔄 File tail

### Q: ClickHouse 需要什么版本？为什么需要密码？

**A**: 
- 推荐 ClickHouse 20.x+，已测试 25.11.2.24
- ClickHouse 25.x 版本强制要求密码认证（安全增强）
- 可以通过环境变量 `CLICKHOUSE_PASSWORD` 或配置文件设置密码

### Q: 如何设置 ClickHouse 密码？

**A**: 两种方式：

1. **环境变量**（推荐）：
```bash
export CLICKHOUSE_PASSWORD=your_password
docker run -e CLICKHOUSE_PASSWORD=your_password ...
```

2. **配置文件**：
在 `filebeat-to-ck/config.yaml` 中设置：
```yaml
clickhouse:
  password: "your_password"
```

### Q: 数据丢失怎么办？

**A**: 
- 转换器使用 ClickHouse 的 JSONEachRow 格式，这是最可靠的接口
- 如果写入失败，会记录详细错误日志
- 建议：
  - 监控转换器日志
  - 设置告警机制
  - 定期检查数据完整性
  - 使用消息队列作为缓冲层（未来支持）

### Q: 如何验证数据是否正确写入？

**A**: 
```bash
# 查询数据总数
docker exec clickhouse clickhouse-client --password your_password \
  --query "SELECT count() FROM logs.logs_table"

# 查询最新数据
docker exec clickhouse clickhouse-client --password your_password \
  --query "SELECT timestamp, message, container FROM logs.logs_table ORDER BY timestamp DESC LIMIT 10"

# 查询特定时间段的数据
docker exec clickhouse clickhouse-client --password your_password \
  --query "SELECT count() FROM logs.logs_table WHERE timestamp >= '2025-12-04 10:00:00'"
```

### Q: 如何扩展转换器以支持更多输入源？

**A**: 
- 转换器框架已预留扩展接口
- 在 `config.yaml` 中配置新的输入源
- 实现对应的处理函数（参考 `handleBulk`、`handleLogstash` 等）
- 添加相应的依赖库（如 Kafka、Redis 客户端）

### Q: 转换器可以部署多个实例吗？

**A**: 
- 可以！转换器是无状态设计
- 使用负载均衡器（如 Nginx）分发请求
- 每个实例独立连接到 ClickHouse
- 建议使用 Docker Compose 的 `scale` 功能

## 🛠️ 开发指南

### 项目结构

```
.
├── filebeat-to-ck/              # 转换器代码
│   ├── main.go                  # 主程序
│   ├── config.yaml              # 配置文件
│   ├── Dockerfile               # Docker 构建文件
│   ├── go.mod                   # Go 依赖
│   ├── go.sum                   # Go 依赖校验
│   └── README.md                # 转换器说明
├── docker-compose.yml            # Docker Compose 配置
├── filebeat.yml                 # Filebeat 配置
├── clickhouse-init/              # ClickHouse 初始化脚本
│   └── init.sql
└── README.md                    # 本文档
```

### 构建

```bash
cd filebeat-to-ck
go mod download
go build -o filebeat-to-ck main.go
```

### 测试

```bash
# 单元测试
go test ./...

# 集成测试
docker-compose up -d

# 发送测试数据
curl -X POST http://localhost:8080/filebeat \
  -H "Content-Type: application/json" \
  -d '{"@timestamp":"2025-12-04T10:00:00Z","message":"test message","host":{"name":"test"}}'

# 验证数据
docker exec clickhouse clickhouse-client --password your_password \
  --query "SELECT * FROM logs.logs_table ORDER BY timestamp DESC LIMIT 1"
```

### 调试

```bash
# 查看转换器日志
docker logs -f filebeat-to-ck

# 查看详细日志（设置 log_level: debug）
# 编辑 config.yaml，设置 log_level: "debug"

# 测试转换器接口
curl http://localhost:8080/health
```

### 贡献

欢迎提交 Issue 和 Pull Request！

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

## 📄 许可证

MIT License

## 📧 联系方式

- **GitHub**: [shoushinya123/filebeat-to-clickhouse](https://github.com/shoushinya123/filebeat-to-clickhouse)
- **Issues**: [GitHub Issues](https://github.com/shoushinya123/filebeat-to-clickhouse/issues)

## 🙏 致谢

- [Filebeat](https://www.elastic.co/beats/filebeat) - 强大的日志收集工具
- [ClickHouse](https://clickhouse.com/) - 高性能列式数据库
- [Gin](https://gin-gonic.com/) - 轻量级 Go Web 框架

## 📚 相关文档

- [功能实现方式说明](./功能实现方式说明.md)
- [数据流转说明](./数据流转说明.md)
- [验证测试报告](./验证结果-最终成功.md)

---

**⭐ 如果这个项目对你有帮助，请给个 Star！**

