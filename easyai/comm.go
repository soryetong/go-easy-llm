package easyai

import "net/http"

type RoleType string

const (
	IdUser   RoleType = "user"
	IdSystem RoleType = "system"
	IdBot    RoleType = "assistant"
)

type LLMType string

const (
	TypeQWen LLMType = "qwen"
)

type ClientConfig struct {
	Types   LLMType
	Token   string
	baseURL string

	HttpClient *http.Client
}

type ChatRequest struct {
	Model   string         `json:"model"`
	Stream  bool           `json:"stream"`
	Message string         `json:"message"`           // 本轮对话用户输入的内容
	History []*ChatHistory `json:"history,omitempty"` // 上下文历史记录
	Tips    *ChatMessage   `json:"tips,omitempty"`
}

type ChatMessage struct {
	Role    RoleType `json:"role"`
	Content string   `json:"content"`
}

type ChatHistory struct {
	ChatMessage
	CreateTime int64 `json:"create_time"`
}

type ChatResponse struct {
	Role    RoleType `json:"role"`
	Content string   `json:"content"`
}
