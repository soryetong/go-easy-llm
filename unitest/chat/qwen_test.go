package chat

import (
	"context"
	"testing"

	easyllm "github.com/soryetong/go-easy-llm"
	"github.com/soryetong/go-easy-llm/easyai/chatmodule"
)

func TestQWenNormalChat(t *testing.T) {
	globalParams := new(chatmodule.QWenParameters)
	globalParams.Input = &chatmodule.QWenInputMessages{}
	tipsMsg := &chatmodule.ChatMessage{Role: chatmodule.IdSystem, Content: "You are a helpful assistant,你的名字是xx,由XX自主研发的AI助手"}
	globalParams.Input.Messages = append(globalParams.Input.Messages, tipsMsg)
	globalParams.Parameters = map[string]interface{}{
		"temperature": 0.8,
		"top_p":       0.8,
		"max_tokens":  1500,
	}

	config := easyllm.DefaultConfig("your-token", chatmodule.ChatTypeQWen)
	client := easyllm.NewChatClient(config).SetGlobalParams(globalParams)
	resp, err := client.NormalChat(context.Background(), &chatmodule.ChatRequest{
		Model:   "qwen-plus",
		Message: "介绍一下自己",
	})
	if err != nil {
		t.Log(err)
		return
	}

	t.Log("resp", resp)
}

func TestQWenStreamChat(t *testing.T) {
	globalParams := new(chatmodule.QWenParameters)
	globalParams.Input = &chatmodule.QWenInputMessages{}
	tipsMsg := &chatmodule.ChatMessage{Role: chatmodule.IdSystem, Content: "You are a helpful assistant,你的名字是xx,由XX自主研发的AI助手"}
	globalParams.Input.Messages = append(globalParams.Input.Messages, tipsMsg)

	config := easyllm.DefaultConfig("your-token", chatmodule.ChatTypeQWen)
	client := easyllm.NewChatClient(config)
	client.SetCustomParams(globalParams)
	resp, err := client.StreamChat(context.Background(), &chatmodule.ChatRequest{
		Model:   "qwen-plus",
		Message: "介绍一下你自己",
	})
	if err != nil {
		t.Log(err)
		return
	}

	// markdownFilterSrv := new(service.MarkdownProcessor)
	for content := range resp {
		t.Log("content: ", content)

		// if markdownFilterSrv.Do(content.Content) != "" {
		// 	t.Log("content.Content", content.Content)
		// }
	}
}
