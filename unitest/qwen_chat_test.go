package unitest

import (
	"context"
	easyllm "github.com/soryetong/go-easy-llm"
	"github.com/soryetong/go-easy-llm/easyai"
	"testing"
)

func TestQWenNormalChat(t *testing.T) {
	globalParams := new(easyai.QWenParameters)
	globalParams.Input = &easyai.QWenInputMessages{}
	tipsMsg := &easyai.ChatMessage{Role: easyai.IdSystem, Content: "You are a helpful assistant,你的名字是xx,由XX自主研发的AI助手"}
	globalParams.Input.Messages = append(globalParams.Input.Messages, tipsMsg)
	globalParams.Parameters = map[string]interface{}{
		"temperature": 0.8,
		"top_p":       0.8,
		"max_tokens":  1500,
	}

	config := easyllm.DefaultConfig("your-token", easyai.TypeQWen)
	client := easyllm.NewClient(config).SetGlobalParams(globalParams)
	resp, reply, err := client.NormalChat(context.Background(), &easyai.ChatRequest{
		Model:   easyai.QWenTurboModel,
		Message: "",
	})
	if err != nil {
		t.Log(err)
		return
	}

	t.Log("resp", resp)
	t.Log("reply-RequestId", reply.(*easyai.QWenResponse).RequestId)
}

func TestQWenStreamChat(t *testing.T) {
	globalParams := new(easyai.QWenParameters)
	globalParams.Input = &easyai.QWenInputMessages{}
	tipsMsg := &easyai.ChatMessage{Role: easyai.IdSystem, Content: "You are a helpful assistant,你的名字是xx,由XX自主研发的AI助手"}
	globalParams.Input.Messages = append(globalParams.Input.Messages, tipsMsg)

	config := easyllm.DefaultConfig("your-token", easyai.TypeQWen)
	client := easyllm.NewClient(config)
	client.SetCustomParams(globalParams)
	resp, err := client.StreamChat(context.Background(), &easyai.ChatRequest{
		Model:   easyai.QWenTurboModel,
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