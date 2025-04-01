package chatmodule

import (
	"context"
	"fmt"
	"io"

	"github.com/soryetong/go-easy-llm/utils"
)

// DouBaoParameters 豆包大模型参数
// url: https://www.volcengine.com/docs/82379/1298454
type DouBaoParameters struct {
	Model            string                 `json:"model"`
	Messages         []*ChatMessage         `json:"messages"` // 可以携带tips、history、input(本轮输入的内容,必要)
	Stream           bool                   `json:"stream"`
	StreamOptions    StreamOptions          `json:"stream_options,omitempty"`
	ExtraHeaders     map[string]interface{} `json:"extra_headers,omitempty"`
	MaxTokens        int                    `json:"max_tokens,omitempty"`
	ServiceTier      string                 `json:"service_tier,omitempty"` // 指定是否使用TPM保障包
	Stop             []string               `json:"stop,omitempty"`         // 模型遇到 stop 字段所指定的字符串时将停止继续生成
	FrequencyPenalty float64                `json:"frequency_penalty,omitempty"`
	PresencePenalty  float64                `json:"presence_penalty,omitempty"`
	Temperature      float64                `json:"temperature,omitempty"`
	TopP             float64                `json:"top_p,omitempty"`
	Logprobs         bool                   `json:"logprobs,omitempty"`
	TopLogprobs      int                    `json:"top_logprobs,omitempty"`
	LogitBias        map[string]int         `json:"logit_bias,omitempty"`
}

type DouBaoChat struct {
	BaseChat
}

func NewDouBaoChat(config *ClientConfig) *DouBaoChat {
	baseUrl := "https://ark.cn-beijing.volces.com/api/v3/chat/completions"
	if config.baseURL != "" {
		baseUrl = config.baseURL
	}

	return &DouBaoChat{
		BaseChat: BaseChat{
			typeName:     "[豆包]",
			baseUrl:      baseUrl,
			defaultModel: "",
			Config:       config,
			globalParams: new(DouBaoParameters),
		},
	}
}

func (self *DouBaoChat) GetRequestParams() (*DouBaoParameters, error) {
	messages, err := self.setCommonParamsAndMessages(true)
	if err != nil {
		return nil, err
	}

	var requestParams = new(DouBaoParameters)
	if self.globalParams != nil {
		var ok bool
		requestParams, ok = self.globalParams.(*DouBaoParameters)
		if !ok {
			return nil, fmt.Errorf("SetCustomParamsMustBe: [DouBaoParameters]")
		}
	}

	requestParams.Model = self.requestModel
	requestParams.Messages = append(requestParams.Messages, messages...)
	if requestParams.Temperature == 0 {
		requestParams.Temperature = 0.8
	}

	return requestParams, nil
}

func (self *DouBaoChat) NormalChat(ctx context.Context, request *ChatRequest) (*ChatResponse, error) {
	self.request = request
	respBody, err := self.doHttpRequest(false)
	if err != nil {
		return nil, err
	}

	return self.processNormalSuccessResp(ctx, respBody, self.openAiNormalResponse)
}

func (self *DouBaoChat) StreamChat(ctx context.Context, request *ChatRequest) (<-chan *ChatResponse, error) {
	self.request = request
	respBody, err := self.doHttpRequest(true)
	if err != nil {
		return nil, err
	}

	messageChan := make(chan *ChatResponse)
	utils.GoSafe(func() {
		self.processStreamResponse(self.getJobCtx(), respBody, messageChan, self.openAiStreamResponse)
	})

	return messageChan, nil
}

func (self *DouBaoChat) doHttpRequest(isStream bool) (respBody io.ReadCloser, errMsg error) {
	params, err := self.GetRequestParams()
	params.Stream = isStream
	if err != nil {
		errMsg = fmt.Errorf("GetRequestParamsFailed: %v", err)
		return
	}

	return self.doOpenAiHttpRequest(params)
}
