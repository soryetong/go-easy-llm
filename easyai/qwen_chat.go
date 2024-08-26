package easyai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
)

const (
	ChatModelQWenTurbo = "qwen-turbo"

	QWenBaseUrl = "https://dashscope.aliyuncs.com/api/v1/services/aigc/text-generation/generation"
)

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
	Output    *QWenOutput `json:"output"`
	Usage     *QWenUsage  `json:"usage"`
	RequestId string      `json:"request_id"`
}

type QWenOutput struct {
	Choices []*QWenChoices `json:"choices,omitempty"`
}

type QWenChoices struct {
	Message      *ChatMessage `json:"message"`
	FinishReason string       `json:"finish_reason"`
}

type QWenUsage struct {
	TotalTokens  int64 `json:"total_tokens"`
	InputTokens  int64 `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
}

type QWenChat struct {
	Config *ClientConfig
	Params *QWenParameters

	request     *ChatRequest
	paramsClone *QWenParameters
}

func (self *QWenChat) SetCustomParams(params interface{}) {
	marshal, err := json.Marshal(params)
	if err != nil {
		errMsg := fmt.Errorf("设置全局参数-序列化失败: { %w }", err)
		_, _ = fmt.Fprintf(os.Stderr, "\n\n [go-easy-llm] \n  %v \n\n", errMsg)

		return
	}

	self.Params = &QWenParameters{}
	if err = json.Unmarshal(marshal, self.Params); err != nil {
		errMsg := fmt.Errorf("设置全局参数-反序列化失败: { %w }", err)
		_, _ = fmt.Fprintf(os.Stderr, "\n\n [go-easy-llm] \n  %v \n\n", errMsg)

		return
	}
}

func (self *QWenChat) NormalChat(ctx context.Context, request *ChatRequest) (*ChatResponse, interface{}, error) {
	if err := self.checkAndSetRequest(request); err != nil {
		errMsg := fmt.Errorf("调用通义千问API-参数不合法: { %w }", err)
		_, _ = fmt.Fprintf(os.Stderr, "\n\n [go-easy-llm] \n  %v \n\n", errMsg)

		return nil, nil, errMsg
	}

	respBody, err := self.doHttpRequest()
	if err != nil {
		errMsg := fmt.Errorf("调用通义千问API失败: { %w }", err)
		_, _ = fmt.Fprintf(os.Stderr, "\n\n [go-easy-llm] \n  %v \n\n", errMsg)

		return nil, nil, errMsg
	}

	defer respBody.Close()
	respByte, err := io.ReadAll(respBody)
	if err != nil {
		errMsg := fmt.Errorf("调用通义千问API-解析响应数据失败: { %w }", err)
		_, _ = fmt.Fprintf(os.Stderr, "\n\n [go-easy-llm] \n  %v \n\n", errMsg)

		return nil, nil, errMsg
	}

	var output = new(QWenResponse)
	if err = json.Unmarshal(respByte, &output); err != nil {
		errMsg := fmt.Errorf("调用通义千问API-结果反序列化失败: { %w }", err)
		_, _ = fmt.Fprintf(os.Stderr, "\n\n [go-easy-llm] \n  %v \n\n", errMsg)

		return nil, nil, errMsg
	}

	respMsg := new(ChatResponse)
	if len(output.Output.Choices) > 0 {
		respMsg.Content = output.Output.Choices[0].Message.Content
		respMsg.Role = output.Output.Choices[0].Message.Role
	}

	return respMsg, output, nil
}

func (self *QWenChat) StreamChat(ctx context.Context, request *ChatRequest) (<-chan *ChatResponse, error) {
	request.Stream = true
	if err := self.checkAndSetRequest(request); err != nil {
		errMsg := fmt.Errorf("调用通义千问API-参数不合法: { %w }", err)
		_, _ = fmt.Fprintf(os.Stderr, "\n\n [go-easy-llm] \n  %v \n\n", errMsg)

		return nil, errMsg
	}

	respBody, err := self.doHttpRequest()
	if err != nil {
		errMsg := fmt.Errorf("调用通义千问API失败: { %w }", err)
		_, _ = fmt.Fprintf(os.Stderr, "\n\n [go-easy-llm] \n  %v \n\n", errMsg)

		return nil, errMsg
	}

	messageChan := make(chan *ChatResponse)
	go func() {
		info := ""
		reader := bufio.NewReader(respBody)
		for {
			line, readErr := reader.ReadString('\n')
			if readErr != nil {
				_, _ = fmt.Fprintf(os.Stderr, "\n\n [go-easy-llm] \n "+
					"  流式解析数据结束, 原因: { %v } \n\n", readErr)
				close(messageChan)
				return
			}

			if line[len(line)-1] == '\n' {
				line = line[:len(line)-1]
			}
			if len(line) > 5 && line[:5] == "data:" {
				var result QWenResponse
				_ = json.Unmarshal([]byte(line[5:]), &result)
				for _, choice := range result.Output.Choices {
					if choice.Message.Role == IdBot && choice.Message.Content != info {
						respMsg := new(ChatResponse)
						respMsg.Role = choice.Message.Role
						respMsg.Content = choice.Message.Content
						messageChan <- respMsg
					}
				}
			}
		}
	}()

	return messageChan, nil
}

func (self *QWenChat) checkAndSetRequest(request *ChatRequest) error {
	if request == nil || request.Message == "" {
		return errors.New("message不能为空")
	}

	self.request = new(ChatRequest)
	self.request = request
	self.cloneParams()

	return nil
}

func (self *QWenChat) cloneParams() {
	self.paramsClone = new(QWenParameters)
	self.setParamsModel()
	self.setParamsInput()
	self.setParamsParameters()
}

func (self *QWenChat) setParamsModel() {
	defaultModel := self.request.Model
	if self.request.Model == "" {
		defaultModel = ChatModelQWenTurbo
	}

	if self.paramsClone.Model == "" {
		self.paramsClone.Model = defaultModel
	}
}

func (self *QWenChat) setParamsInput() {
	if self.Params == nil {
		self.Params = new(QWenParameters)
		self.Params.Input = new(QWenInputMessages)
	}

	self.paramsClone.Input = new(QWenInputMessages)
	if self.Params.Input != nil && len(self.Params.Input.Messages) >= 0 {
		for _, message := range self.Params.Input.Messages {
			self.paramsClone.Input.Messages = append(self.paramsClone.Input.Messages, message)
		}
	}
	self.paramsClone.Input.Messages = append(self.paramsClone.Input.Messages, &ChatMessage{
		Role:    IdUser,
		Content: self.request.Message,
	})
	if self.request.Tips != nil {
		self.paramsClone.Input.Messages = append(self.paramsClone.Input.Messages, self.request.Tips)
	}

	if len(self.request.History) > 0 {
		for _, history := range self.request.History {
			self.paramsClone.Input.Messages = append(self.paramsClone.Input.Messages, &ChatMessage{
				Role:    history.Role,
				Content: history.Content,
			})
		}
	}
}

func (self *QWenChat) setParamsParameters() {
	if self.Params.Parameters != nil {
		self.paramsClone.Parameters = self.Params.Parameters
	} else {
		self.paramsClone.Parameters = map[string]interface{}{
			"temperature": 0.8,
			"top_p":       0.8,
			"max_tokens":  1500,
		}
	}

	// 强制返回output.choices字段
	self.paramsClone.Parameters["result_format"] = "message"

	if self.request.Stream {
		self.paramsClone.Parameters["incremental_output"] = true
	}
}

func (self *QWenChat) doHttpRequest() (respBody io.ReadCloser, errMsg error) {
	respBody = nil
	jsonBody, err := json.Marshal(self.paramsClone)
	if err != nil {
		errMsg = fmt.Errorf("序列化请求参数失败, 原因: %w", err)
		return
	}

	req, err := http.NewRequest("POST", QWenBaseUrl, bytes.NewReader(jsonBody))
	if err != nil {
		errMsg = fmt.Errorf("构造http请求失败, 原因: %w", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", self.Config.Token))
	if self.request.Stream {
		req.Header.Set("X-DashScope-SSE", "enable")
	}

	resp, err := self.Config.HttpClient.Do(req)
	if err != nil {
		errMsg = fmt.Errorf("http请求失败, 原因: %w", err)
		return
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		var errResp QWenResponseError
		b, _ := io.ReadAll(resp.Body)
		if err = json.Unmarshal(b, &errResp); err != nil {
			errMsg = fmt.Errorf("http结果序列化失败, 原因: %v", err)
			return
		}

		errMsg = fmt.Errorf("http请求失败, 原因: %s, message: %s", errResp.Code, errResp.Message)
		return
	}

	respBody = resp.Body

	return
}
