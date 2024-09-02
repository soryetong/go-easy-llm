package easyai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/soryetong/go-easy-llm/utils"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type HunYuanRegionType string

const (
	ChatModelHunYuanPro          = "hunyuan-pro"
	ChatModelHunYuanStandard     = "hunyuan-standard"
	ChatModelHunYuanLite         = "hunyuan-lite"
	ChatModelHunYuanRole         = "hunyuan-role"
	ChatModelHunYuanFunctionCall = "hunyuan-functioncall"
	ChatModelHunYuanCode         = "hunyuan-code"

	HunYuanBaseUrl       = "https://hunyuan.tencentcloudapi.com"
	HunYuanHost          = "hunyuan.tencentcloudapi.com"
	HunYuanDefaultAction = "ChatCompletions"

	HunYuanRegionGuangZhou HunYuanRegionType = "ap-guangzhou"
	HunYuanRegionBeijing   HunYuanRegionType = "ap-beijing"
	HunYuanRegionShangHai  HunYuanRegionType = "ap-shanghai"
)

type HunYuanParameters struct {
	Model             string              `json:"Model"`
	Version           string              `json:"Version,omitempty"`
	Messages          []*ChatMessageUpper `json:"Messages"`
	Stream            bool                `json:"Stream,omitempty"`
	StreamModeration  bool                `json:"StreamModeration,omitempty"`
	TopP              float64             `json:"TopP,omitempty"`        // 不建议修改
	Temperature       float64             `json:"Temperature,omitempty"` // 不建议修改
	Citation          bool                `json:"Citation,omitempty"`
	EnableSpeedSearch bool                `json:"EnableSpeedSearch,omitempty"`
	Language          string              `json:"Language,omitempty"` // 默认为zh-CN, 仅部分接口支持
}

type HunYuanResponse struct {
	Response HunYuanResponseNormalData `json:"Response"`
}

type HunYuanResponseNormalData struct {
	HunYuanResponseData
	Choices []*HunYuanChoices `json:"Choices"`
}

type HunYuanResponseStreamData struct {
	HunYuanResponseData
	Choices []*HunYuanStreamChoices `json:"Choices"`
}

type HunYuanResponseData struct {
	Error     *HunYuanError `json:"Error"`
	RequestId string        `json:"RequestId"`
	Note      string        `json:"Note"`
	Id        string        `json:"Id"`
	Usage     *HunYuanUsage `json:"Usage"`
	Created   int64         `json:"Created"`
}

type HunYuanError struct {
	Code    string `json:"Code"`
	Message string `json:"Message"`
}

type HunYuanUsage struct {
	PromptTokens     int64 `json:"PromptTokens"`
	CompletionTokens int64 `json:"CompletionTokens"`
	TotalTokens      int64 `json:"TotalTokens"`
}

type HunYuanChoices struct {
	Message      *ChatMessageUpper `json:"Message"`
	FinishReason string            `json:"FinishReason"`
}

type HunYuanStreamChoices struct {
	Delta        *ChatMessageUpper `json:"Delta"`
	FinishReason string            `json:"FinishReason"`
}

type HunYuanChat struct {
	Config *ClientConfig
	Params *HunYuanParameters

	request     *ChatRequest
	paramsClone *HunYuanParameters
}

func (self *HunYuanChat) SetCustomParams(params interface{}) {
	marshal, err := json.Marshal(params)
	if err != nil {
		errMsg := fmt.Errorf("混元大模型-设置全局参数-序列化失败: { %w }", err)
		_, _ = fmt.Fprintf(os.Stderr, "\n\n [go-easy-llm] \n  %v \n\n", errMsg)

		return
	}

	self.Params = &HunYuanParameters{}
	if err = json.Unmarshal(marshal, self.Params); err != nil {
		errMsg := fmt.Errorf("混元大模型-设置全局参数-反序列化失败: { %w }", err)
		_, _ = fmt.Fprintf(os.Stderr, "\n\n [go-easy-llm] \n  %v \n\n", errMsg)

		return
	}
}

func (self *HunYuanChat) NormalChat(ctx context.Context, request *ChatRequest) (*ChatResponse, interface{}, error) {
	if err := self.checkAndSetRequest(request); err != nil {
		errMsg := fmt.Errorf("调用混元API-参数不合法: { %w }", err)
		_, _ = fmt.Fprintf(os.Stderr, "\n\n [go-easy-llm] \n  %v \n\n", errMsg)

		return nil, nil, errMsg
	}

	respBody, err := self.doHttpRequest()
	defer respBody.Close()
	if err != nil {
		errMsg := fmt.Errorf("调用混元API失败: { %w }", err)
		_, _ = fmt.Fprintf(os.Stderr, "\n\n [go-easy-llm] \n  %v \n\n", errMsg)

		return nil, nil, errMsg
	}

	respByte, err := io.ReadAll(respBody)
	if err != nil {
		errMsg := fmt.Errorf("调用混元API-解析响应数据失败: { %w }", err)
		_, _ = fmt.Fprintf(os.Stderr, "\n\n [go-easy-llm] \n  %v \n\n", errMsg)

		return nil, nil, errMsg
	}

	var output = new(HunYuanResponse)
	if err = json.Unmarshal(respByte, &output); err != nil {
		errMsg := fmt.Errorf("调用混元API-结果反序列化失败: { %w }", err)
		_, _ = fmt.Fprintf(os.Stderr, "\n\n [go-easy-llm] \n  %v \n\n", errMsg)

		return nil, nil, errMsg
	}

	if output.Response.Error != nil {
		errMsg := fmt.Errorf("调用混元API失败: %s", output.Response.Error.Message)
		_, _ = fmt.Fprintf(os.Stderr, "\n\n [go-easy-llm] \n  %v \n\n", errMsg)

		return nil, nil, errMsg
	}

	respMsg := new(ChatResponse)
	if len(output.Response.Choices) > 0 {
		respMsg.Role = output.Response.Choices[0].Message.Role
		respMsg.Content = output.Response.Choices[0].Message.Content
	}

	return respMsg, output, nil
}

func (self *HunYuanChat) StreamChat(ctx context.Context, request *ChatRequest) (<-chan *ChatResponse, error) {
	request.Stream = true
	if err := self.checkAndSetRequest(request); err != nil {
		errMsg := fmt.Errorf("调用混元API-参数不合法: { %w }", err)
		_, _ = fmt.Fprintf(os.Stderr, "\n\n [go-easy-llm] \n  %v \n\n", errMsg)

		return nil, errMsg
	}

	respBody, err := self.doHttpRequest()
	if err != nil {
		errMsg := fmt.Errorf("调用混元API失败: { %w }", err)
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
				var result HunYuanResponseStreamData
				_ = json.Unmarshal([]byte(line[5:]), &result)
				if result.Error != nil {
					errMsg := fmt.Errorf("调用混元API失败: %s", result.Error.Message)
					_, _ = fmt.Fprintf(os.Stderr, "\n\n [go-easy-llm] \n  %v \n\n", errMsg)

					close(messageChan)
					return
				}

				for _, choice := range result.Choices {
					if choice.Delta.Role == IdBot && choice.Delta.Content != info {
						respMsg := new(ChatResponse)
						respMsg.Role = choice.Delta.Role
						respMsg.Content = choice.Delta.Content
						messageChan <- respMsg
					}
				}
			}
		}
	}()

	return messageChan, err
}

func (self *HunYuanChat) checkAndSetRequest(request *ChatRequest) error {
	if request == nil || request.Message == "" {
		return errors.New("message不能为空")
	}

	self.request = new(ChatRequest)
	self.request = request
	self.cloneParams()

	return nil
}

func (self *HunYuanChat) cloneParams() {
	self.paramsClone = new(HunYuanParameters)
	self.setParamsModel()
	self.setParamsInput()
	self.setParamsParameters()
}

func (self *HunYuanChat) setParamsModel() {
	defaultModel := self.request.Model
	if self.request.Model == "" {
		defaultModel = ChatModelHunYuanPro
	}

	if self.paramsClone.Model == "" {
		self.paramsClone.Model = defaultModel
	}
}

func (self *HunYuanChat) setParamsInput() {
	if self.Params == nil {
		self.Params = new(HunYuanParameters)
	}

	if len(self.Params.Messages) > 0 {
		for _, message := range self.Params.Messages {
			self.paramsClone.Messages = append(self.paramsClone.Messages, message)
		}
	}
	self.paramsClone.Messages = append(self.paramsClone.Messages, &ChatMessageUpper{
		Role:    IdUser,
		Content: self.request.Message,
	})
	if self.request.Tips != nil {
		self.paramsClone.Messages = append(self.paramsClone.Messages, &ChatMessageUpper{
			Role:    IdSystem,
			Content: self.request.Tips.Content,
		})
	}

	if len(self.request.History) > 0 {
		for _, history := range self.request.History {
			self.paramsClone.Messages = append(self.paramsClone.Messages, &ChatMessageUpper{
				Role:    history.Role,
				Content: history.Content,
			})
		}
	}
}

func (self *HunYuanChat) setParamsParameters() {
	if self.Params.Model != "" {
		self.paramsClone = self.Params
	} else {
		self.paramsClone.Version = "2023-09-01"
		self.paramsClone.Language = "zh-CN"
	}

	if self.request.Stream {
		self.paramsClone.Stream = true
		self.paramsClone.StreamModeration = true
	}
}

func (self *HunYuanChat) doHttpRequest() (respBody io.ReadCloser, errMsg error) {
	respBody = nil
	xTcVersion := self.paramsClone.Version
	language := self.paramsClone.Language

	self.paramsClone.Version = ""  // 不参与加密
	self.paramsClone.Language = "" // 不参与加密
	jsonBody, err := json.Marshal(self.paramsClone)
	if err != nil {
		errMsg = fmt.Errorf("序列化请求参数失败, 原因: %w", err)
		return
	}

	req, err := http.NewRequest("POST", HunYuanBaseUrl, bytes.NewReader(jsonBody))
	if err != nil {
		errMsg = fmt.Errorf("构造http请求失败, 原因: %w", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", self.getAuthorization(string(jsonBody)))
	req.Header.Set("X-TC-Action", HunYuanDefaultAction)
	req.Header.Set("X-TC-Version", xTcVersion)
	req.Header.Set("X-TC-Language", language)
	req.Header.Set("Host", HunYuanHost)
	req.Header.Set("X-TC-Timestamp", fmt.Sprintf("%d", time.Now().Unix()))

	resp, err := self.Config.HttpClient.Do(req)
	if err != nil {
		errMsg = fmt.Errorf("http请求失败, 原因: %w", err)
		return
	}

	respBody = resp.Body

	return
}

func (self *HunYuanChat) getAuthorization(payload string) (authorization string) {
	algorithm := "TC3-HMAC-SHA256"
	timestamp := time.Now().Unix()
	service := "hunyuan"

	// 拼接canonical请求参数
	httpRequestMethod := "POST"
	canonicalURI := "/"
	canonicalQueryString := ""
	canonicalHeaders := fmt.Sprintf("content-type:%s\nhost:%s\nx-tc-action:%s\n",
		"application/json", HunYuanHost, strings.ToLower(HunYuanDefaultAction))
	signedHeaders := "content-type;host;x-tc-action"
	hashedRequestPayload := utils.Sha256hex(payload)
	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		httpRequestMethod,
		canonicalURI,
		canonicalQueryString,
		canonicalHeaders,
		signedHeaders,
		hashedRequestPayload)

	// 构建sign签名
	date := time.Unix(timestamp, 0).UTC().Format("2006-01-02")
	credentialScope := fmt.Sprintf("%s/%s/tc3_request", date, service)
	hashedCanonicalRequest := utils.Sha256hex(canonicalRequest)
	string2sign := fmt.Sprintf("%s\n%d\n%s\n%s",
		algorithm,
		timestamp,
		credentialScope,
		hashedCanonicalRequest)

	// 签名字符串加密
	secretDate := utils.HmacSha256(date, "TC3"+self.Config.SecretKey)
	secretService := utils.HmacSha256(service, secretDate)
	secretSigning := utils.HmacSha256("tc3_request", secretService)
	signature := hex.EncodeToString([]byte(utils.HmacSha256(string2sign, secretSigning)))

	// 组装authorization
	authorization = fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		algorithm,
		self.Config.SecretId,
		credentialScope,
		signedHeaders,
		signature)

	return
}
