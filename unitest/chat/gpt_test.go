package chat

import (
	"context"
	"testing"

	easyllm "github.com/soryetong/go-easy-llm"
	"github.com/soryetong/go-easy-llm/easyai/chatmodule"
)

func TestGPTNormalChat(t *testing.T) {
	config := easyllm.DefaultConfigWithProxy("your-api-key", chatmodule.ChatTypeGPT, "http://127.0.0.1:7890")
	client := easyllm.NewChatClient(config)
	resp, err := client.NormalChat(context.Background(), &chatmodule.ChatRequest{
		Model:   "gpt-4o-mini",
		Message: "介绍一下你自己",
	})
	if err != nil {
		t.Log(err)
		return
	}

	t.Log("resp", resp)
}

func TestGPTStreamChat(t *testing.T) {
	config := easyllm.DefaultConfigWithProxy("your-api-key", chatmodule.ChatTypeGPT, "http://127.0.0.1:7890")
	client := easyllm.NewChatClient(config)
	resp, err := client.StreamChat(context.Background(), &chatmodule.ChatRequest{
		Model:   "gpt-4o-mini",
		Message: "介绍一下你自己",
	})
	if err != nil {
		t.Log(err)
		return
	}

	for content := range resp {
		t.Log("content: ", content)
	}
}
