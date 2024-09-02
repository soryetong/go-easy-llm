package unitest

import (
	"context"
	easyllm "github.com/soryetong/go-easy-llm"
	"github.com/soryetong/go-easy-llm/easyai"
	"testing"
)

func TestHunYuanNormalChat(t *testing.T) {
	config := easyllm.DefaultConfigWithSecret("your-secretId", "your-secretKey", easyai.ChatTypeHunYuan)
	client := easyllm.NewChatClient(config)
	resp, reply, err := client.NormalChat(context.Background(), &easyai.ChatRequest{
		Model:   easyai.ChatModelHunYuanPro,
		Message: "介绍一下你自己",
	})
	if err != nil {
		t.Log(err)
		return
	}

	t.Log("resp", resp)
	t.Log("reply", reply.(*easyai.HunYuanResponse))
}

func TestHunYuanStreamChat(t *testing.T) {
	config := easyllm.DefaultConfigWithSecret("your-secretId", "your-secretKey", easyai.ChatTypeHunYuan)
	client := easyllm.NewChatClient(config)
	resp, err := client.StreamChat(context.Background(), &easyai.ChatRequest{
		Model:   easyai.ChatModelHunYuanPro,
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
