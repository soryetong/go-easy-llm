package chat

import (
	"context"
	"time"

	easyllm "github.com/soryetong/go-easy-llm"
	"github.com/soryetong/go-easy-llm/easyai/chatmodule"

	"testing"
)

func TestHunYuanNormalChat(t *testing.T) {
	config := easyllm.DefaultConfigWithSecret("your-secretId", "your-secretKey", chatmodule.ChatTypeHunYuan)
	client := easyllm.NewChatClient(config)
	resp, err := client.NormalChat(context.Background(), &chatmodule.ChatRequest{
		Model:   "hunyuan-pro",
		Message: "介绍一下你自己",
	})
	if err != nil {
		t.Log(err)
		return
	}

	t.Log("resp", resp)
}

func TestHunYuanStreamChat(t *testing.T) {
	globalParams := new(chatmodule.HunYuanParameters)
	tipsMsg := &chatmodule.ChatMessageUpper{Role: chatmodule.IdSystem, Content: "You are a helpful assistant,你的名字是张三,由喜羊羊自主研发的AI助手"}
	globalParams.Messages = append(globalParams.Messages, tipsMsg)
	config := easyllm.DefaultConfigWithSecret("your-secretId", "your-secretKey", chatmodule.ChatTypeHunYuan)
	client := easyllm.NewChatClient(config).SetGlobalParams(globalParams)
	resp, err := client.StreamChat(context.Background(), &chatmodule.ChatRequest{
		Model:   "hunyuan-pro",
		Message: "你是谁",
	})
	if err != nil {
		t.Log(err)
		return
	}

	var sessionId string

	// 模拟停止
	go func() {
		time.Sleep(time.Second * 2)
		client.Stop(context.Background(), sessionId)
	}()

	for content := range resp {
		if content == nil {
			break
		}

		sessionId = content.SessionId
		t.Log("content: ", content)
	}
}
