package chatmodule

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/soryetong/go-easy-llm/utils"
)

// QWenParameters 这是最终的请求参数
// url: https://help.aliyun.com/zh/model-studio/developer-reference/use-qwen-by-calling-api
type QWenParameters struct {
	Model      string                 `json:"model"`
	Input      *QWenInputMessages     `json:"input"`
	Parameters map[string]interface{} `json:"parameters"`
}

type QWenInputMessages struct {
	Messages []*ChatMessage `json:"messages"`
}

type QWenResponseError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestId string `json:"request_id"`
}

type QWenResponse struct {
	Output *QWenOutput `json:"output"`
}

type QWenOutput struct {
	Choices []*QWenChoices `json:"choices,omitempty"`
}

type QWenChoices struct {
	Message      *ChatMessage `json:"message"`
	FinishReason string       `json:"finish_reason"`
}

type QWenChat struct {
	BaseChat
}

func NewQWenChat(config *ClientConfig) *QWenChat {
	baseUrl := "https://dashscope.aliyuncs.com/api/v1/services/aigc/text-generation/generation"
	if config.baseURL != "" {
		baseUrl = config.baseURL
	}

	return &QWenChat{
		BaseChat: BaseChat{
			typeName:     "[通义千问]",
			baseUrl:      baseUrl,
			defaultModel: "qwen-turbo",
			Config:       config,
			globalParams: new(QWenParameters),
		},
	}
}

func (self *QWenChat) GetRequestParams() (*QWenParameters, error) {
	messages, err := self.setCommonParamsAndMessages(true)
	if err != nil {
		return nil, err
	}

	var requestParams = new(QWenParameters)
	if self.globalParams != nil {
		var ok bool
		requestParams, ok = self.globalParams.(*QWenParameters)
		if !ok {
			return nil, fmt.Errorf("SetCustomParamsMustBe: [QWenParameters]")
		}
	}

	requestParams.Model = self.requestModel
	requestParams.Input.Messages = append(requestParams.Input.Messages, messages...)
	if requestParams.Parameters == nil {
		requestParams.Parameters = map[string]interface{}{
			"temperature": 0.8,
			"top_p":       0.8,
			"max_tokens":  1500,
		}
	}
	// 强制返回output.choices字段
	requestParams.Parameters["result_format"] = "message"

	return requestParams, nil
}

func (self *QWenChat) NormalChat(ctx context.Context, request *ChatRequest) (*ChatResponse, error) {
	self.request = request
	respBody, err := self.doHttpRequest(false)
	if err != nil {
		return nil, err
	}

	return self.processNormalSuccessResp(ctx, respBody, self.normalResponse)
}

func (self *QWenChat) normalResponse(data string) (*ChatResponse, error) {
	var output = new(QWenResponse)
	if err := json.Unmarshal([]byte(data), output); err != nil {
		return nil, fmt.Errorf("ApiResultDeserializationFailed: %v", err)
	}

	respMsg := new(ChatResponse)
	if len(output.Output.Choices) > 0 {
		respMsg.Role = output.Output.Choices[0].Message.Role
		respMsg.Content = output.Output.Choices[0].Message.Content
	}

	return respMsg, nil
}

func (self *QWenChat) StreamChat(ctx context.Context, request *ChatRequest) (<-chan *ChatResponse, error) {
	self.request = request
	respBody, err := self.doHttpRequest(true)
	if err != nil {
		return nil, err
	}

	messageChan := make(chan *ChatResponse)
	utils.GoSafe(func() {
		self.processStreamResponse(self.getJobCtx(), respBody, messageChan, self.streamResponse)
	})

	return messageChan, nil
}

func (self *QWenChat) streamResponse(data, lastMessage string) (*ChatResponse, string, error) {
	var result QWenResponse
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return nil, "", fmt.Errorf("ApiResultDeserializationFailed: %v", err)
	}

	for _, choice := range result.Output.Choices {
		if choice.Message.Role == IdBot && choice.Message.Content != lastMessage {
			resp := &ChatResponse{
				Role:    choice.Message.Role,
				Content: choice.Message.Content,
			}
			return resp, resp.Content, nil
		}
	}

	return nil, "", nil
}

func (self *QWenChat) doHttpRequest(isStream bool) (io.ReadCloser, error) {
	params, err := self.GetRequestParams()
	if err != nil {
		return nil, fmt.Errorf("GetRequestParamsFailed: %v", err)
	}

	var headers = make(map[string]string)
	headers["Authorization"] = fmt.Sprintf("Bearer %s", self.Config.Token)
	if isStream {
		headers["X-DashScope-SSE"] = "enable"
		params.Parameters["incremental_output"] = true
	}

	resp, err := self.doCommonHttpRequest(params, headers)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		var errResp QWenResponseError
		bodyBytes, _ := io.ReadAll(resp.Body)
		if err = json.Unmarshal(bodyBytes, &errResp); err != nil {
			return nil, fmt.Errorf("HttpResultSerializationFailed: %v", err)
		}

		utils.Logger.Error(self.typeName+"-http请求失败", "resp", errResp)
		return nil, fmt.Errorf("HttpRequestError: %s", errResp.Message)
	}

	return resp.Body, nil
}
