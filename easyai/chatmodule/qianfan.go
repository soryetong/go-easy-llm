package chatmodule

import (
	"context"
	"fmt"
	"io"

	"github.com/soryetong/go-easy-llm/utils"
)

// QianFanParameters 千帆大模型参数
// uri: https://cloud.baidu.com/doc/WENXINWORKSHOP/s/Fm2vrveyu
type QianFanParameters struct {
	Model               string         `json:"model"`
	Messages            []*ChatMessage `json:"messages"` // 可以携带tips、history、input(本轮输入的内容,必要)
	Stream              bool           `json:"stream"`
	StreamOptions       StreamOptions  `json:"stream_options,omitempty"`
	Temperature         float64        `json:"temperature,omitempty"`
	TopP                float64        `json:"top_p,omitempty"`
	PenaltyScore        float64        `json:"penalty_score,omitempty"`
	MaxCompletionTokens int            `json:"max_completion_tokens,omitempty"`
	Seed                int            `json:"seed,omitempty"`
	Stop                []string       `json:"stop,omitempty"` // 模型遇到 stop 字段所指定的字符串时将停止继续生成
	User                string         `json:"user,omitempty"`
	FrequencyPenalty    float64        `json:"frequency_penalty,omitempty"`
	PresencePenalty     float64        `json:"presence_penalty,omitempty"`
}

type QianFanChat struct {
	BaseChat
}

func NewQianFanChat(config *ClientConfig) *QianFanChat {
	baseUrl := "https://qianfan.baidubce.com/v2/chat/completions"
	if config.baseURL != "" {
		baseUrl = config.baseURL
	}

	return &QianFanChat{
		BaseChat: BaseChat{
			typeName:     "[千帆]",
			baseUrl:      baseUrl,
			defaultModel: "ernie-4.5-8k-preview",
			Config:       config,
			globalParams: new(QianFanParameters),
		},
	}
}

func (self *QianFanChat) GetRequestParams() (*QianFanParameters, error) {
	messages, err := self.setCommonParamsAndMessages(true)
	if err != nil {
		return nil, err
	}

	var requestParams = new(QianFanParameters)
	if self.globalParams != nil {
		var ok bool
		requestParams, ok = self.globalParams.(*QianFanParameters)
		if !ok {
			return nil, fmt.Errorf("SetCustomParamsMustBe: [QianFanParameters]")
		}
	}

	requestParams.Model = self.requestModel
	requestParams.Messages = append(requestParams.Messages, messages...)
	if requestParams.Temperature == 0 {
		requestParams.Temperature = 0.8
	}

	return requestParams, nil
}

func (self *QianFanChat) NormalChat(ctx context.Context, request *ChatRequest) (*ChatResponse, error) {
	self.request = request
	respBody, err := self.doHttpRequest(false)
	if err != nil {
		return nil, err
	}

	return self.processNormalSuccessResp(ctx, respBody, self.openAiNormalResponse)
}

func (self *QianFanChat) StreamChat(ctx context.Context, request *ChatRequest) (<-chan *ChatResponse, error) {
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

func (self *QianFanChat) doHttpRequest(isStream bool) (respBody io.ReadCloser, errMsg error) {
	params, err := self.GetRequestParams()
	params.Stream = isStream
	if err != nil {
		errMsg = fmt.Errorf("GetRequestParamsFailed: %v", err)
		return
	}

	return self.doOpenAiHttpRequest(params)
}
