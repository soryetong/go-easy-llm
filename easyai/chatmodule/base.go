package chatmodule

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/soryetong/go-easy-llm/service"
	"github.com/soryetong/go-easy-llm/utils"
)

type RoleType string

const (
	IdUser   RoleType = "user"
	IdSystem RoleType = "system"
	IdBot    RoleType = "assistant"
)

type LLMType string

const (
	ChatTypeQWen    LLMType = "qwen"
	ChatTypeHunYuan LLMType = "hunyuan"
	ChatTypeGPT     LLMType = "gpt"
	ChatTypeDouBao  LLMType = "doubao"
	ChatTypeQianFan LLMType = "qianfan"
)

type ClientConfig struct {
	Types     LLMType
	Token     string
	SecretId  string
	SecretKey string
	baseURL   string

	HttpClient *http.Client
}

type ChatRequest struct {
	Url          string         `json:"url,omitempty"`     // 请求地址, 当为空时，使用官方的URL
	Model        string         `json:"model"`             // 请求模型
	Message      string         `json:"message"`           // 本轮对话用户输入的内容
	History      []*ChatHistory `json:"history,omitempty"` // 上下文历史记录
	Tips         *ChatMessage   `json:"tips,omitempty"`    // 提示词
	NeedMetadata bool           `json:"need_metadata"`     // 是否需要返回metadata(大模型的元数据)
	SessionId    string         `json:"session_id"`        // 会话ID
}

type StreamOptions struct {
	IncludeUsage bool `json:"include_usage"` // 是否包含本次请求的 token 用量统计信息
}

type ChatMessage struct {
	Role    RoleType `json:"role"`
	Content string   `json:"content"`
}

type ChatMessageUpper struct {
	Role    RoleType `json:"Role"`
	Content string   `json:"Content"`
}

type ChatHistory struct {
	ChatMessage
	CreateTime int64 `json:"create_time"`
}

type ChatResponse struct {
	Role      RoleType `json:"role"`
	Content   string   `json:"content"`
	Metadata  string   `json:"metadata"` // 大模型返回的元数据
	SessionId string   `json:"session_id"`
}

/** ----------------------------------------------- API 响应 ----------------------------------------------- */
// ApiSuccessResponse 是 openai 的通用响应结构体
type ApiSuccessResponse struct {
	Choices []struct {
		Message *ChatMessage `json:"message"`
	} `json:"choices"`
}

type ApiSuccessStreamResponse struct {
	Id      string                  `json:"id"`
	Created int64                   `json:"created"`
	Choices []StreamResponseChoices `json:"choices"`
}

type StreamResponseChoices struct {
	Delta *ChatMessage `json:"delta"`
}

type ApiErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

/** ----------------------------------------------- BaseChat ----------------------------------------------- */
// BaseChat 是所有 LLM 模型的基类，实现了一些通用的方法和属性
type BaseChat struct {
	typeName     string
	baseUrl      string
	defaultModel string
	request      *ChatRequest
	Config       *ClientConfig

	requestUrl   string      // 请求地址, 当request.Url为空时，使用baseUrl
	requestModel string      // 请求模型, 当request.Model为空时，使用defaultModel
	globalParams interface{} // 全局参数
}

func (self *BaseChat) SetCustomParams(params interface{}) {
	marshal, err := json.Marshal(params)
	if err != nil {
		utils.Logger.Error(self.typeName+"-[SetCustomParams] 设置全局参数:序列化失败",
			"err", err, "params", params)
		return
	}

	if err = json.Unmarshal(marshal, self.globalParams); err != nil {
		utils.Logger.Error(self.typeName+"-[SetCustomParams] 设置全局参数:反序列化失败",
			"err", err, "params", params, "marshal", string(marshal))
		return
	}
}

func (self *BaseChat) Stop(ctx context.Context, sessionId string) {
	cancel := service.GetChatSession(sessionId)
	if cancel != nil {
		cancel()
	}
	service.RemoveChatSession(sessionId)
}

func (self *BaseChat) getJobCtx() context.Context {
	sessionId := self.request.SessionId
	if sessionId == "" {
		sessionId = service.GenerateSessionId()
	}
	ctx := context.WithValue(context.Background(), "sessionId", sessionId)
	jobCtx, cancel := context.WithCancel(ctx)
	service.SetChatSession(sessionId, cancel)

	return jobCtx
}

func (self *BaseChat) setCommonParamsAndMessages(setMessages bool) ([]*ChatMessage, error) {
	if self.request == nil || self.request.Message == "" {
		return nil, errors.New("MessageCannotBeEmpty")
	}

	self.requestUrl = self.baseUrl
	if self.request.Url != "" {
		self.requestUrl = self.request.Url
	}

	self.requestModel = self.defaultModel
	if self.request.Model != "" {
		self.requestModel = self.request.Model
	}

	if setMessages == false {
		return nil, nil
	}

	messages := make([]*ChatMessage, 0)
	if self.request.Tips != nil {
		messages = append(messages, &ChatMessage{
			Role:    IdSystem,
			Content: self.request.Tips.Content,
		})
	}

	if len(self.request.History) > 0 {
		for _, history := range self.request.History {
			messages = append(messages, &ChatMessage{
				Role:    history.Role,
				Content: history.Content,
			})
		}
	}

	messages = append(messages, &ChatMessage{
		Role:    IdUser,
		Content: self.request.Message,
	})

	return messages, nil
}

// doOpenAiHttpRequest 发起http请求, openai系列的通用请求方式
func (self *BaseChat) doOpenAiHttpRequest(params interface{}) (io.ReadCloser, error) {
	var headers = make(map[string]string)
	headers["Authorization"] = fmt.Sprintf("Bearer %s", self.Config.Token)
	resp, err := self.doCommonHttpRequest(params, headers)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		var errResp ApiErrorResponse
		bodyBytes, _ := io.ReadAll(resp.Body)
		if err = json.Unmarshal(bodyBytes, &errResp); err != nil {
			return nil, fmt.Errorf("HttpResultSerializationFailed: %v", err)
		}

		utils.Logger.Error(self.typeName+"-http请求失败", "resp", errResp)
		return nil, fmt.Errorf("HttpRequestError: %s", errResp.Error.Code)
	}

	return resp.Body, nil
}

// doCommonHttpRequest 发起http请求, 不同厂商的特色请求方式
func (self *BaseChat) doCommonHttpRequest(params interface{}, headers map[string]string) (*http.Response, error) {
	jsonBody, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("HttpBodySerializationFailed: %v", err)
	}

	req, err := http.NewRequest("POST", self.requestUrl, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("NewHttpRequestError: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := self.Config.HttpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HttpRequestError: %v", err)
	}

	return resp, nil
}

type NormalParseFunc func(string) (*ChatResponse, error)

// processNormalSuccessResp 处理正常响应 不同厂商的通用处理方式
func (self *BaseChat) processNormalSuccessResp(ctx context.Context, respBody io.ReadCloser, parseFunc NormalParseFunc) (*ChatResponse, error) {
	defer respBody.Close()
	respByte, err := io.ReadAll(respBody)
	if err != nil {
		return nil, fmt.Errorf("ReadApiResultFailed: %v", err)
	}

	messages, err := parseFunc(string(respByte))
	if err != nil {
		return nil, err
	}
	if self.request.NeedMetadata {
		messages.Metadata = string(respByte)
	}

	return messages, nil
}

func (self *BaseChat) openAiNormalResponse(data string) (*ChatResponse, error) {
	var output = new(ApiSuccessResponse)
	if err := json.Unmarshal([]byte(data), &output); err != nil {
		return nil, fmt.Errorf("ApiResultDeserializationFailed: %v", err)
	}

	respMsg := new(ChatResponse)
	if len(output.Choices) > 0 {
		respMsg.Role = output.Choices[0].Message.Role
		respMsg.Content = output.Choices[0].Message.Content
	}

	return respMsg, nil
}

type StreamParseFunc func(string, string) (*ChatResponse, string, error)

// processStreamResponse 处理流式响应 不同厂商的通用处理方式
func (self *BaseChat) processStreamResponse(ctx context.Context, respBody io.ReadCloser, messageChan chan *ChatResponse, parseFunc StreamParseFunc) {
	defer close(messageChan)
	defer respBody.Close()

	sessionId, _ := ctx.Value("sessionId").(string)
	scanner := bufio.NewScanner(respBody)
	var lastMessage string
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			utils.Logger.Info(self.typeName+"-[StreamChat] 用户主动终止会话", "sessionId", sessionId)
			return
		default:
			line := scanner.Text()
			if strings.HasPrefix(line, "data:") {
				jsonData := strings.TrimSpace(line[5:])
				if jsonData == "[DONE]" {
					break
				}

				messages, newLastMessage, err := parseFunc(jsonData, lastMessage)
				if err != nil {
					utils.Logger.Error(self.typeName+"-[StreamChat] 解析消息失败", "err", err, "jsonData", jsonData)
					continue
				}

				if messages == nil {
					continue
				}

				lastMessage = newLastMessage
				if self.request.NeedMetadata {
					messages.Metadata = jsonData
				}
				messages.SessionId = sessionId
				messageChan <- messages
			}
		}
	}

	service.RemoveChatSession(sessionId)
	if err := scanner.Err(); err != nil {
		utils.Logger.Error(self.typeName+"-[StreamChat] 读取流数据失败", "err", err)
	}
}

func (self *BaseChat) openAiStreamResponse(data, lastMessage string) (*ChatResponse, string, error) {
	var result ApiSuccessStreamResponse
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return nil, "", fmt.Errorf("DeserializationFailed: %v", err)
	}

	for _, choice := range result.Choices {
		if choice.Delta.Content != "" && choice.Delta.Content != lastMessage {
			lastMessage = choice.Delta.Content
			resp := &ChatResponse{
				Role:    choice.Delta.Role,
				Content: choice.Delta.Content,
			}
			return resp, resp.Content, nil
		}
	}

	return nil, "", nil
}
