-- 创建数据库
CREATE DATABASE IF NOT EXISTS logs;

-- 创建表用于接收 Filebeat 日志
-- 使用灵活的表结构，支持 Filebeat 输出的 JSON 字段
-- 注意：字段名使用点号分隔，在 ClickHouse 中需要用反引号包裹
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
    `raw_json` String DEFAULT ''  -- 存储完整的 JSON 字符串（可选）
)
ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (timestamp)
SETTINGS index_granularity = 8192;

