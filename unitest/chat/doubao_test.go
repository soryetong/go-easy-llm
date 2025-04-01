package chat

import (
	"context"
	"testing"
	"time"

	easyllm "github.com/soryetong/go-easy-llm"
	"github.com/soryetong/go-easy-llm/easyai/chatmodule"
)

func TestDouBaoNormalChat(t *testing.T) {
	globalParams := new(chatmodule.DouBaoParameters)
	tipsMsg := &chatmodule.ChatMessage{Role: chatmodule.IdSystem, Content: "You are a helpful assistant,你的名字是张三,由喜羊羊自主研发的AI助手"}
	globalParams.Messages = append(globalParams.Messages, tipsMsg)
	config := easyllm.DefaultConfig("your-token", chatmodule.ChatTypeDouBao)
	client := easyllm.NewChatClient(config).SetGlobalParams(globalParams)
	resp, err := client.NormalChat(context.Background(), &chatmodule.ChatRequest{
		Model:        "ep-20250324152958-s4nwn",
		Message:      "介绍一下你自己",
		NeedMetadata: true,
	})
	if err != nil {
		t.Log(err, "----------")
		return
	}

	t.Log("resp", resp)
}

func TestDouBaoStreamChat(t *testing.T) {
	config := easyllm.DefaultConfig("your-token", chatmodule.ChatTypeDouBao)
	client := easyllm.NewChatClient(config)
	resp, err := client.StreamChat(context.Background(), &chatmodule.ChatRequest{
		Model:   "ep-20250324152958-s4nwn",
		Message: "我需要你回复一个1分钟的内容",
	})
	if err != nil {
		t.Log(err)
		return
	}

	var sessionId string

	// 模拟停止
	go func() {
		time.Sleep(time.Second * 5)
		t.Log("stop")
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
