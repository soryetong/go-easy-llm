package chatmodule

import (
	"context"
	"fmt"
	"io"

	"github.com/soryetong/go-easy-llm/utils"
)

type GPTParameters struct {
	Model            string         `json:"model"`
	Messages         []*ChatMessage `json:"messages"`
	Temperature      float64        `json:"temperature"`
	TopP             float64        `json:"top_p,omitempty"`
	Stream           bool           `json:"stream"`
	N                int            `json:"n,omitempty"`
	MaxTokens        int            `json:"max_tokens,omitempty"`
	PresencePenalty  float64        `json:"presence_penalty,omitempty"`
	FrequencyPenalty float64        `json:"frequency_penalty,omitempty"`
	LogitBias        map[string]int `json:"logit_bias,omitempty"`
}

type GPTChat struct {
	BaseChat
}

func NewGPTChat(config *ClientConfig) *GPTChat {
	baseUrl := "https://api.openai.com/v1/chat/completions"
	if config.baseURL != "" {
		baseUrl = config.baseURL
	}

	return &GPTChat{
		BaseChat: BaseChat{
			typeName:     "[ChatGPT]",
			baseUrl:      baseUrl,
			defaultModel: "gpt-4o-mini",
			Config:       config,
			globalParams: new(GPTParameters),
		},
	}
}

func (self *GPTChat) GetRequestParams() (*GPTParameters, error) {
	messages, err := self.setCommonParamsAndMessages(true)
	if err != nil {
		return nil, err
	}

	var requestParams = new(GPTParameters)
	if self.globalParams != nil {
		var ok bool
		requestParams, ok = self.globalParams.(*GPTParameters)
		if !ok {
			return nil, fmt.Errorf("SetCustomParamsMustBe: [GPTParameters]")
		}
	}

	requestParams.Model = self.requestModel
	requestParams.Messages = append(requestParams.Messages, messages...)
	if requestParams.Temperature == 0 {
		requestParams.Temperature = 0.8
	}

	return requestParams, nil
}

func (self *GPTChat) NormalChat(ctx context.Context, request *ChatRequest) (*ChatResponse, error) {
	self.request = request
	respBody, err := self.doHttpRequest(false)
	if err != nil {
		return nil, err
	}

	return self.processNormalSuccessResp(ctx, respBody, self.openAiNormalResponse)
}

func (self *GPTChat) StreamChat(ctx context.Context, request *ChatRequest) (<-chan *ChatResponse, error) {
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

func (self *GPTChat) doHttpRequest(isStream bool) (respBody io.ReadCloser, errMsg error) {
	params, err := self.GetRequestParams()
	params.Stream = isStream
	if err != nil {
		errMsg = fmt.Errorf("GetRequestParamsFailed: %v", err)
		return
	}

	return self.doOpenAiHttpRequest(params)
}
