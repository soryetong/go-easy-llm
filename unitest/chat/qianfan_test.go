package chat

import (
	"context"
	"testing"

	easyllm "github.com/soryetong/go-easy-llm"
	"github.com/soryetong/go-easy-llm/easyai/chatmodule"
)

func TestQianFanNormalChat(t *testing.T) {
	globalParams := new(chatmodule.QianFanParameters)
	tipsMsg := &chatmodule.ChatMessage{Role: chatmodule.IdSystem, Content: "You are a helpful assistant,你的名字是张三,由喜羊羊自主研发的AI助手"}
	globalParams.Messages = append(globalParams.Messages, tipsMsg)
	config := easyllm.DefaultConfig("your-token", chatmodule.ChatTypeQianFan)
	client := easyllm.NewChatClient(config).SetGlobalParams(globalParams)
	resp, err := client.NormalChat(context.Background(), &chatmodule.ChatRequest{
		Model:   "ernie-4.5-8k-preview",
		Message: "介绍一下你自己",
	})
	if err != nil {
		t.Log(err, "----------")
		return
	}

	t.Log("resp", resp)
}

func TestQianFanStreamChat(t *testing.T) {
	config := easyllm.DefaultConfig("your-token", chatmodule.ChatTypeQianFan)
	client := easyllm.NewChatClient(config)
	resp, err := client.StreamChat(context.Background(), &chatmodule.ChatRequest{
		Model:   "ernie-4.5-8k-preview",
		Message: "简单的介绍一下你自己",
	})
	if err != nil {
		t.Log(err)
		return
	}

	for content := range resp {
		t.Log("content: ", content)
	}
}
