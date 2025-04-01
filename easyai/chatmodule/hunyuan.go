package chatmodule

import (
	"bufio"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/soryetong/go-easy-llm/service"
	"github.com/soryetong/go-easy-llm/utils"
)

const (
	HunYuanBaseUrl       = "https://hunyuan.tencentcloudapi.com"
	HunYuanHost          = "hunyuan.tencentcloudapi.com"
	HunYuanDefaultAction = "ChatCompletions"
)

// HunYuanParameters 混元大模型参数
// url: https://cloud.tencent.com/document/product/1729/105701
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

/* ----------------------- 普通返回 -----------------------*/
// HunYuanNormalResponse 混元大模型返回
type HunYuanNormalResponse struct {
	Response HunYuanNormalResponseData `json:"Response"`
}

type HunYuanNormalResponseData struct {
	HunYuanError
	Choices []*HunYuanNormalChoices `json:"Choices"`
}

type HunYuanNormalChoices struct {
	Message      *ChatMessageUpper `json:"Message"`
	FinishReason string            `json:"FinishReason"`
}

/* ----------------------- 流式返回 -----------------------*/
// HunYuanStreamResponse 混元大模型流式返回
type HunYuanStreamResponse struct {
	Response HunYuanStreamResponseData `json:"Response"`
}

type HunYuanStreamResponseData struct {
	HunYuanError
	Choices []*HunYuanStreamChoices `json:"Choices"`
}

type HunYuanStreamChoices struct {
	Delta        *ChatMessageUpper `json:"Delta"`
	FinishReason string            `json:"FinishReason"`
}

/* ----------------------- 错误信息 -----------------------*/
// HunYuanError 混元大模型错误信息
type HunYuanError struct {
	Error *HunYuanErrorInfo `json:"Error"`
}

type HunYuanErrorInfo struct {
	Code    string `json:"Code"`
	Message string `json:"Message"`
}

type HunYuanChat struct {
	BaseChat
}

func NewHunYuanChat(config *ClientConfig) *HunYuanChat {
	baseUrl := HunYuanBaseUrl
	if config.baseURL != "" {
		baseUrl = config.baseURL
	}

	return &HunYuanChat{
		BaseChat: BaseChat{
			typeName:     "[混元]",
			baseUrl:      baseUrl,
			defaultModel: "hunyuan-pro",
			Config:       config,
			globalParams: new(HunYuanParameters),
		},
	}
}

func (self *HunYuanChat) GetRequestParams() (*HunYuanParameters, error) {
	messages, err := self.setCommonParamsAndMessages(true)
	if err != nil {
		return nil, err
	}

	var requestParams = new(HunYuanParameters)
	if self.globalParams != nil {
		var ok bool
		requestParams, ok = self.globalParams.(*HunYuanParameters)
		if !ok {
			return nil, fmt.Errorf("SetCustomParamsMustBe: [QWenParameters]")
		}
	}

	requestParams.Model = self.requestModel
	messageUpper := make([]*ChatMessageUpper, 0)
	for _, message := range messages {
		messageUpper = append(messageUpper, &ChatMessageUpper{
			Role:    message.Role,
			Content: message.Content,
		})
	}
	requestParams.Messages = append(requestParams.Messages, messageUpper...)
	if requestParams.Version == "" {
		requestParams.Version = "2023-09-01"
	}
	if requestParams.Language == "" {
		requestParams.Language = "zh-CN"
	}

	return requestParams, nil
}

func (self *HunYuanChat) NormalChat(ctx context.Context, request *ChatRequest) (*ChatResponse, error) {
	self.request = request
	respBody, err := self.doHttpRequest(false)
	if err != nil {
		return nil, err
	}

	return self.processNormalSuccessResp(ctx, respBody, self.normalResponse)
}

func (self *HunYuanChat) normalResponse(data string) (*ChatResponse, error) {
	var output = new(HunYuanNormalResponse)
	if err := json.Unmarshal([]byte(data), output); err != nil {
		return nil, fmt.Errorf("ApiResultDeserializationFailed: %v", err)
	}

	if output.Response.Error != nil {
		utils.Logger.Error(self.typeName+"-http请求失败", "resp", output.Response)
		return nil, fmt.Errorf("ApiResultDeserializationFailed: %v", output.Response.Error.Code)
	}

	respMsg := new(ChatResponse)
	if len(output.Response.Choices) > 0 {
		respMsg.Role = output.Response.Choices[0].Message.Role
		respMsg.Content = output.Response.Choices[0].Message.Content
	}

	return respMsg, nil
}

func (self *HunYuanChat) StreamChat(ctx context.Context, request *ChatRequest) (<-chan *ChatResponse, error) {
	self.request = request
	respBody, err := self.doHttpRequest(true)
	if err != nil {
		return nil, err
	}

	jobCtx := self.getJobCtx()
	sessionId, _ := jobCtx.Value("sessionId").(string)

	messageChan := make(chan *ChatResponse)
	utils.GoSafe(func() {
		defer close(messageChan)
		defer respBody.Close()

		scanner := bufio.NewScanner(respBody)
		var lastMessage string
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				utils.Logger.Info(self.typeName+"-[StreamChat] 用户主动终止会话", "sessionId", sessionId)
				return
			default:
				line := scanner.Text()
				if strings.Contains(line, "Error") {
					var result HunYuanStreamResponse
					if err = json.Unmarshal([]byte(line), &result); err != nil {
						utils.Logger.Error(self.typeName+"-[StreamChat] 解析Error消息失败", "err", err, "line", line)
						break
					}

					if result.Response.Error != nil {
						utils.Logger.Error(self.typeName+"-[StreamChat] 请求失败错", "err", result.Response.Error.Message)
						break
					}
				}

				if strings.HasPrefix(line, "data:") {
					jsonData := strings.TrimSpace(line[5:])
					if jsonData == "[DONE]" {
						break
					}

					var result HunYuanStreamResponseData
					if err = json.Unmarshal([]byte(jsonData), &result); err != nil {
						utils.Logger.Error(self.typeName+"-[StreamChat] 解析消息失败", "err", err, "jsonData", jsonData)
						continue
					}

					for _, choice := range result.Choices {
						if choice.Delta.Role == IdBot && choice.Delta.Content != lastMessage {
							lastMessage = choice.Delta.Content
							messages := &ChatResponse{
								Role:    choice.Delta.Role,
								Content: choice.Delta.Content,
							}
							if self.request.NeedMetadata {
								messages.Metadata = jsonData
							}
							messages.SessionId = sessionId
							messageChan <- messages
						}
					}
				}
			}
		}

		service.RemoveChatSession(sessionId)
		if err = scanner.Err(); err != nil {
			utils.Logger.Error(self.typeName+"-[StreamChat] 读取流数据失败", "err", err)
		}
	})

	return messageChan, nil
}

func (self *HunYuanChat) doHttpRequest(isStream bool) (io.ReadCloser, error) {
	params, err := self.GetRequestParams()
	if err != nil {
		return nil, fmt.Errorf("GetRequestParamsFailed: %v", err)
	}

	params.Stream = isStream
	xTcVersion := params.Version
	language := params.Language
	params.Version = ""  // 不参与加密
	params.Language = "" // 不参与加密
	jsonBody, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("HttpBodySerializationFailed: %v", err)
	}

	var headers = make(map[string]string)
	headers["Authorization"] = self.getAuthorization(string(jsonBody))
	headers["X-TC-Action"] = HunYuanDefaultAction
	headers["X-TC-Version"] = xTcVersion
	headers["X-TC-Language"] = language
	headers["Host"] = HunYuanHost
	headers["X-TC-Timestamp"] = fmt.Sprintf("%d", time.Now().Unix())
	resp, err := self.doCommonHttpRequest(params, headers)
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
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
