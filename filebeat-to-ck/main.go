package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

// Config 配置结构
type Config struct {
	Server struct {
		Host string `yaml:"host"`
		Port int    `yaml:"port"`
	} `yaml:"server"`
	ClickHouse struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		Database string `yaml:"database"`
		Table    string `yaml:"table"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
	} `yaml:"clickhouse"`
	Inputs struct {
		Elasticsearch struct {
			Enabled bool `yaml:"enabled"`
			Port    int  `yaml:"port"`
		} `yaml:"elasticsearch"`
		Logstash struct {
			Enabled  bool   `yaml:"enabled"`
			Port     int    `yaml:"port"`
			Protocol string `yaml:"protocol"`
		} `yaml:"logstash"`
		Kafka struct {
			Enabled    bool     `yaml:"enabled"`
			Brokers    []string `yaml:"brokers"`
			Topics     []string `yaml:"topics"`
			GroupID    string   `yaml:"group_id"`
			AutoCommit bool     `yaml:"auto_commit"`
		} `yaml:"kafka"`
		Redis struct {
			Enabled  bool   `yaml:"enabled"`
			Address  string `yaml:"address"`
			Password string `yaml:"password"`
			Mode     string `yaml:"mode"` // list or pubsub
			Key      string `yaml:"key"`
		} `yaml:"redis"`
		File struct {
			Enabled bool     `yaml:"enabled"`
			Paths   []string `yaml:"paths"`
			Follow  bool     `yaml:"follow"`
		} `yaml:"file"`
		TCP struct {
			Enabled bool   `yaml:"enabled"`
			Port    int    `yaml:"port"`
			Format  string `yaml:"format"`
		} `yaml:"tcp"`
	} `yaml:"inputs"`
	LogLevel string `yaml:"log_level"`
}

// FilebeatEvent Filebeat 事件结构
type FilebeatEvent struct {
	Timestamp interface{}            `json:"@timestamp"` // 可能是字符串或时间对象
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Container map[string]interface{} `json:"container,omitempty"`
	Host      map[string]interface{} `json:"host,omitempty"`
	Docker    map[string]interface{} `json:"docker,omitempty"`
	Agent     map[string]interface{} `json:"agent,omitempty"`
	Log       map[string]interface{} `json:"log,omitempty"`
	// 支持任意其他字段
	Extra map[string]interface{} `json:"-"`
}

// GetTimestamp 获取时间戳
func (e *FilebeatEvent) GetTimestamp() time.Time {
	if e.Timestamp == nil {
		return time.Now()
	}
	
	switch v := e.Timestamp.(type) {
	case string:
		// 尝试多种时间格式
		formats := []string{
			time.RFC3339,
			time.RFC3339Nano,
			"2006-01-02T15:04:05.000Z",
			"2006-01-02T15:04:05Z",
			"2006-01-02 15:04:05",
		}
		for _, format := range formats {
			if t, err := time.Parse(format, v); err == nil {
				return t
			}
		}
		return time.Now()
	case time.Time:
		return v
	default:
		return time.Now()
	}
}

// ElasticsearchBulkRequest Elasticsearch bulk API 请求格式
type ElasticsearchBulkRequest struct {
	Index struct {
		Index string `json:"_index"`
		Type  string `json:"_type,omitempty"`
		ID    string `json:"_id,omitempty"`
	} `json:"index,omitempty"`
	Create struct {
		Index string `json:"_index"`
		Type  string `json:"_type,omitempty"`
		ID    string `json:"_id,omitempty"`
	} `json:"create,omitempty"`
	Delete struct {
		Index string `json:"_index"`
		Type  string `json:"_type,omitempty"`
		ID    string `json:"_id,omitempty"`
	} `json:"delete,omitempty"`
	Update struct {
		Index string `json:"_index"`
		Type  string `json:"_type,omitempty"`
		ID    string `json:"_id,omitempty"`
	} `json:"update,omitempty"`
	Doc interface{} `json:"doc,omitempty"`
}

var config Config

func main() {
	// 加载配置
	if err := loadConfig(); err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 设置日志级别
	if config.LogLevel == "debug" {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	// 设置 Gin 模式
	if config.LogLevel != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建路由
	r := gin.Default()

	// 健康检查
	r.GET("/health", healthCheck)
	r.GET("/", healthCheck)

	// Elasticsearch 兼容接口（支持 output.elasticsearch）
	r.POST("/_bulk", handleBulk)
	r.POST("/:index/_bulk", handleBulk)
	r.POST("/:index/:type/_bulk", handleBulk)

	// Logstash 兼容接口（支持 output.logstash）
	r.POST("/", handleLogstash)  // Logstash HTTP 输出
	r.Any("/logstash", handleLogstash)

	// 直接 JSON 接收接口（通用）
	r.POST("/events", handleEvents)
	r.POST("/filebeat", handleFilebeat)
	r.POST("/ingest", handleFilebeat)  // 通用接收端点

	// 启动 HTTP 服务器（支持 Elasticsearch 和 Logstash HTTP 输出）
	addr := fmt.Sprintf("%s:%d", config.Server.Host, config.Server.Port)
	log.Printf("🚀 转换器启动在 %s", addr)
	log.Printf("📊 ClickHouse: %s:%d/%s.%s", config.ClickHouse.Host, config.ClickHouse.Port, config.ClickHouse.Database, config.ClickHouse.Table)
	
	// 启动其他输入源
	if config.Inputs.Logstash.Enabled {
		go startLogstashTCP(config.Inputs.Logstash.Port)
	}
	if config.Inputs.Kafka.Enabled {
		go startKafkaConsumer()
	}
	if config.Inputs.Redis.Enabled {
		go startRedisConsumer()
	}
	if config.Inputs.File.Enabled {
		go startFileTail()
	}
	if config.Inputs.TCP.Enabled {
		go startTCPServer()
	}
	
	if err := r.Run(addr); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}

// loadConfig 加载配置文件
func loadConfig() error {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "/etc/filebeat-to-ck/config.yaml"
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 设置默认值
	if config.Server.Host == "" {
		config.Server.Host = "0.0.0.0"
	}
	if config.Server.Port == 0 {
		config.Server.Port = 8080
	}
	if config.ClickHouse.Host == "" {
		config.ClickHouse.Host = "clickhouse"
	}
	if config.ClickHouse.Port == 0 {
		config.ClickHouse.Port = 8123
	}
	if config.ClickHouse.Database == "" {
		config.ClickHouse.Database = "logs"
	}
	if config.ClickHouse.Table == "" {
		config.ClickHouse.Table = "logs_table"
	}

	return nil
}

// healthCheck 健康检查
func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "filebeat-to-clickhouse",
		"time":    time.Now().Format(time.RFC3339),
	})
}

// handleBulk 处理 Elasticsearch bulk API 格式
// Filebeat 发送的格式：每两行一组，第一行是 action，第二行是 document
func handleBulk(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "读取请求体失败"})
		return
	}

	// 解析 bulk 格式（每两行一组：action + document）
	lines := strings.Split(string(body), "\n")
	var events []FilebeatEvent

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		// 尝试解析为 action（index, create, update, delete）
		var action map[string]interface{}
		if err := json.Unmarshal([]byte(line), &action); err != nil {
			// 如果不是有效的 JSON，跳过
			continue
		}

		// 检查是否是 action 行（包含 index, create, update, delete 等键）
		isAction := false
		for key := range action {
			if key == "index" || key == "create" || key == "update" || key == "delete" {
				isAction = true
				break
			}
		}

		// 如果不是 action 行，可能是 document 行（处理异常情况）
		if !isAction {
			// 直接作为 document 处理
			var event FilebeatEvent
			if err := json.Unmarshal([]byte(line), &event); err != nil {
				// 如果解析失败，尝试作为通用 JSON 处理
				var generic map[string]interface{}
				if err := json.Unmarshal([]byte(line), &generic); err == nil {
					event = convertGenericToEvent(generic)
					events = append(events, event)
				}
			} else {
				events = append(events, event)
			}
			continue
		}

		// 是 action 行，下一行应该是 document
		if i+1 < len(lines) {
			i++
			docLine := strings.TrimSpace(lines[i])
			if docLine == "" {
				continue
			}

			var event FilebeatEvent
			if err := json.Unmarshal([]byte(docLine), &event); err != nil {
				// 如果解析失败，尝试作为通用 JSON 处理
				var generic map[string]interface{}
				if err := json.Unmarshal([]byte(docLine), &generic); err == nil {
					event = convertGenericToEvent(generic)
				} else {
					// 解析失败，跳过这个 document
					continue
				}
			}

			events = append(events, event)
		}
	}

	// 批量写入 ClickHouse
	if len(events) > 0 {
		if err := writeToClickHouse(events); err != nil {
			log.Printf("写入 ClickHouse 失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "写入失败"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"took":   len(events),
		"errors": false,
		"items":  len(events),
	})
}

// handleEvents 处理直接 JSON 事件数组
func handleEvents(c *gin.Context) {
	var events []FilebeatEvent
	if err := c.ShouldBindJSON(&events); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 JSON 格式"})
		return
	}

	if err := writeToClickHouse(events); err != nil {
		log.Printf("写入 ClickHouse 失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "写入失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok", "count": len(events)})
}

// handleFilebeat 处理 Filebeat 直接输出
func handleFilebeat(c *gin.Context) {
	var event FilebeatEvent
	if err := c.ShouldBindJSON(&event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 JSON 格式"})
		return
	}

	events := []FilebeatEvent{event}
	if err := writeToClickHouse(events); err != nil {
		log.Printf("写入 ClickHouse 失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "写入失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// convertGenericToEvent 将通用 JSON 转换为事件
func convertGenericToEvent(generic map[string]interface{}) FilebeatEvent {
	event := FilebeatEvent{
		Fields:    make(map[string]interface{}),
		Container: make(map[string]interface{}),
		Host:      make(map[string]interface{}),
		Docker:    make(map[string]interface{}),
		Agent:     make(map[string]interface{}),
		Log:       make(map[string]interface{}),
		Extra:     make(map[string]interface{}),
	}

	// 提取时间戳
	if ts, ok := generic["@timestamp"]; ok {
		event.Timestamp = ts
	}

	// 提取消息
	if msg, ok := generic["message"].(string); ok {
		event.Message = msg
	}

	// 提取其他字段
	for k, v := range generic {
		switch k {
		case "@timestamp", "message":
			continue
		case "container":
			if m, ok := v.(map[string]interface{}); ok {
				event.Container = m
			}
		case "host":
			if m, ok := v.(map[string]interface{}); ok {
				event.Host = m
			}
		case "docker":
			if m, ok := v.(map[string]interface{}); ok {
				event.Docker = m
			}
		case "agent":
			if m, ok := v.(map[string]interface{}); ok {
				event.Agent = m
			}
		case "log":
			if m, ok := v.(map[string]interface{}); ok {
				event.Log = m
			}
		default:
			event.Extra[k] = v
		}
	}

	return event
}

// writeToClickHouse 写入数据到 ClickHouse
func writeToClickHouse(events []FilebeatEvent) error {
	if len(events) == 0 {
		return nil
	}

	// 构建 JSONEachRow 格式的数据
	var jsonLines []string
	for _, event := range events {
		// 构建 ClickHouse 记录
		record := make(map[string]interface{})
		
		// 时间戳
		timestamp := event.GetTimestamp()
		record["timestamp"] = timestamp.Format("2006-01-02 15:04:05")

		// 消息
		record["message"] = event.Message

		// 容器信息
		if event.Container != nil {
			if name, ok := event.Container["name"].(string); ok {
				record["container"] = name
			} else if id, ok := event.Container["id"].(string); ok {
				record["container"] = id
			}
		}

		// 主机信息
		if event.Host != nil {
			if name, ok := event.Host["name"].(string); ok {
				record["host_name"] = name
			}
		}

		// Docker 信息
		if event.Docker != nil {
			if container, ok := event.Docker["container"].(map[string]interface{}); ok {
				if id, ok := container["id"].(string); ok {
					record["docker_container_id"] = id
				}
				if name, ok := container["name"].(string); ok {
					record["docker_container_name"] = name
				}
			}
		}

		// Agent 信息
		if event.Agent != nil {
			if name, ok := event.Agent["name"].(string); ok {
				record["agent_name"] = name
			}
			if version, ok := event.Agent["version"].(string); ok {
				record["agent_version"] = version
			}
		}

		// Log 信息
		if event.Log != nil {
			if path, ok := event.Log["file"].(map[string]interface{}); ok {
				if p, ok := path["path"].(string); ok {
					record["log_file_path"] = p
				}
			}
		}

		// 将整个事件序列化为 JSON 字符串（存储在 raw_json 字段）
		if eventJson, err := json.Marshal(event); err == nil {
			record["raw_json"] = string(eventJson)
		}

		// 转换为 JSON 行
		if jsonBytes, err := json.Marshal(record); err == nil {
			jsonLines = append(jsonLines, string(jsonBytes))
		}
	}

	// 构建 ClickHouse INSERT 请求
	// 使用 URL 编码确保安全
	query := fmt.Sprintf("INSERT INTO %s.%s FORMAT JSONEachRow", config.ClickHouse.Database, config.ClickHouse.Table)
	encodedQuery := url.QueryEscape(query)
	requestURL := fmt.Sprintf("http://%s:%d/?query=%s", config.ClickHouse.Host, config.ClickHouse.Port, encodedQuery)
	
	data := strings.Join(jsonLines, "\n")
	req, err := http.NewRequest("POST", requestURL, bytes.NewBufferString(data))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	// ClickHouse JSONEachRow 格式不需要特定的 Content-Type
	// 但设置一个通用的类型有助于识别
	req.Header.Set("Content-Type", "application/x-ndjson")
	if config.ClickHouse.User != "" {
		req.SetBasicAuth(config.ClickHouse.User, config.ClickHouse.Password)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ClickHouse 返回错误: %d, %s", resp.StatusCode, string(body))
	}

	log.Printf("✅ 成功写入 %d 条记录到 ClickHouse", len(events))
	return nil
}

// handleLogstash 处理 Logstash HTTP 输出
// Filebeat output.logstash 可以配置为 HTTP 输出
func handleLogstash(c *gin.Context) {
	// Logstash HTTP 输出通常是 JSON 格式
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "读取请求体失败"})
		return
	}

	// 尝试解析为单个事件或事件数组
	var events []FilebeatEvent
	
	// 先尝试作为数组解析
	var eventArray []map[string]interface{}
	if err := json.Unmarshal(body, &eventArray); err == nil {
		// 是数组
		for _, item := range eventArray {
			event := convertGenericToEvent(item)
			events = append(events, event)
		}
	} else {
		// 尝试作为单个事件
		var event FilebeatEvent
		if err := json.Unmarshal(body, &event); err == nil {
			events = append(events, event)
		} else {
			// 尝试作为通用 JSON
			var generic map[string]interface{}
			if err := json.Unmarshal(body, &generic); err == nil {
				event := convertGenericToEvent(generic)
				events = append(events, event)
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 JSON 格式"})
				return
			}
		}
	}

	// 写入 ClickHouse
	if len(events) > 0 {
		if err := writeToClickHouse(events); err != nil {
			log.Printf("写入 ClickHouse 失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "写入失败"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok", "count": len(events)})
}

// startLogstashTCP 启动 Logstash TCP 服务器（Lumberjack/Beats protocol）
func startLogstashTCP(port int) {
	log.Printf("📡 启动 Logstash TCP 服务器在端口 %d", port)
	// TODO: 实现 Lumberjack/Beats protocol 支持
	// 这是一个二进制协议，需要专门的库来解析
	log.Printf("⚠️  Logstash TCP 协议支持需要额外的库，当前版本暂不支持")
}

// startKafkaConsumer 启动 Kafka consumer
func startKafkaConsumer() {
	if len(config.Inputs.Kafka.Brokers) == 0 || len(config.Inputs.Kafka.Topics) == 0 {
		log.Printf("⚠️  Kafka 配置不完整，跳过启动")
		return
	}
	log.Printf("📡 启动 Kafka consumer: brokers=%v, topics=%v", config.Inputs.Kafka.Brokers, config.Inputs.Kafka.Topics)
	// TODO: 实现 Kafka consumer
	// 需要使用 kafka-go 库
	log.Printf("⚠️  Kafka consumer 支持需要额外的库，当前版本暂不支持")
}

// startRedisConsumer 启动 Redis consumer
func startRedisConsumer() {
	if config.Inputs.Redis.Address == "" || config.Inputs.Redis.Key == "" {
		log.Printf("⚠️  Redis 配置不完整，跳过启动")
		return
	}
	log.Printf("📡 启动 Redis consumer: address=%s, mode=%s, key=%s", config.Inputs.Redis.Address, config.Inputs.Redis.Mode, config.Inputs.Redis.Key)
	// TODO: 实现 Redis LIST/PUBSUB consumer
	// 需要使用 go-redis 库
	log.Printf("⚠️  Redis consumer 支持需要额外的库，当前版本暂不支持")
}

// startFileTail 启动文件 tail
func startFileTail() {
	if len(config.Inputs.File.Paths) == 0 {
		log.Printf("⚠️  文件路径配置为空，跳过启动")
		return
	}
	log.Printf("📡 启动文件 tail: paths=%v", config.Inputs.File.Paths)
	// TODO: 实现文件 tail
	// 可以使用 fsnotify 库监控文件变化
	log.Printf("⚠️  文件 tail 支持需要额外的库，当前版本暂不支持")
}

// startTCPServer 启动 TCP 服务器
func startTCPServer() {
	if config.Inputs.TCP.Port == 0 {
		log.Printf("⚠️  TCP 端口未配置，跳过启动")
		return
	}
	log.Printf("📡 启动 TCP 服务器在端口 %d, 格式=%s", config.Inputs.TCP.Port, config.Inputs.TCP.Format)
	// TODO: 实现 TCP 服务器
	// 支持 JSON 和 JSON Lines 格式
	log.Printf("⚠️  TCP 服务器支持需要额外实现，当前版本暂不支持")
}

