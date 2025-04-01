package utils

import (
	"encoding/json"
	"log"
	"os"
	"time"
)

var Logger *ILogger

const (
	Red    = "\033[31m"
	Yellow = "\033[33m"
	Green  = "\033[32m"
	Reset  = "\033[0m"
)

// ILogger 结构体封装 log.Logger
type ILogger struct {
	logger *log.Logger
	name   string
}

type LogEntry struct {
	Level     string                 `json:"level"`
	Timestamp string                 `json:"timestamp"`
	Message   string                 `json:"message"`
	Content   map[string]interface{} `json:"content,omitempty"`
}

func NewLogger(output *os.File, name string) *ILogger {
	return &ILogger{
		logger: log.New(output, "", 0),
		name:   name,
	}
}

// Info
func (self *ILogger) Info(msg string, v ...interface{}) {
	log.Println(Green + "[info] [go-easy-llm]: " + msg + Reset)
	self.logWithLevel("info", msg, v...)
}

// Warn
func (self *ILogger) Warn(msg string, v ...interface{}) {
	log.Println(Yellow + "[warn] [go-easy-llm]: " + msg + Reset)
	self.logWithLevel("warn", msg, v...)
}

// Error
func (self *ILogger) Error(msg string, v ...interface{}) {
	log.Println(Red + "[error] [go-easy-llm]: " + msg + Reset)
	self.logWithLevel("error", msg, v...)
}

func (self *ILogger) logWithLevel(level string, msg string, v ...interface{}) {
	if len(v)%2 != 0 {
		self.logger.Println("日志参数必须成对出现！")
		return
	}

	// 解析键值对数据
	content := make(map[string]interface{})
	for i := 0; i < len(v); i += 2 {
		key, ok := v[i].(string)
		if !ok {
			self.logger.Println("日志键必须是字符串类型！")
			return
		}
		content[key] = v[i+1]
	}

	logEntry := LogEntry{
		Level:     level,
		Timestamp: time.Now().Format("2006-01-02 15:04:05.000"),
		Message:   self.name + msg,
		Content:   content,
	}

	jsonData, _ := json.Marshal(logEntry)
	self.logger.Println(string(jsonData))
}
