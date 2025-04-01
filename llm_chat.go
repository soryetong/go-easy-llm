package easyllm

import (
	"context"
	"os"

	"github.com/soryetong/go-easy-llm/easyai/chatmodule"
	"github.com/soryetong/go-easy-llm/utils"
)

type ChatClient struct {
	*chatmodule.ClientConfig
	LLMChatInterface
}

type LLMChatInterface interface {
	SetCustomParams(params interface{})

	NormalChat(ctx context.Context, request *chatmodule.ChatRequest) (*chatmodule.ChatResponse, error)
	StreamChat(ctx context.Context, request *chatmodule.ChatRequest) (<-chan *chatmodule.ChatResponse, error)
	Stop(ctx context.Context, sessionId string)
}

func NewChatClient(config *chatmodule.ClientConfig) *ChatClient {
	utils.Logger = utils.NewLogger(os.Stdout, "[go-easy-llm:]")

	return &ChatClient{
		config,
		getLLM(config),
	}
}

func (c *ChatClient) SetGlobalParams(params interface{}) *ChatClient {
	c.SetCustomParams(params)

	return c
}

func getLLM(cfg *chatmodule.ClientConfig) LLMChatInterface {
	switch cfg.Types {
	case chatmodule.ChatTypeQWen:
		return chatmodule.NewQWenChat(cfg)
	case chatmodule.ChatTypeHunYuan:
		return chatmodule.NewHunYuanChat(cfg)
	case chatmodule.ChatTypeGPT:
		return chatmodule.NewGPTChat(cfg)
	case chatmodule.ChatTypeDouBao:
		return chatmodule.NewDouBaoChat(cfg)
	case chatmodule.ChatTypeQianFan:
		return chatmodule.NewQianFanChat(cfg)
	default:
		utils.Logger.Error("获取Client异常: 无效的LLM配置", "nowTypes", cfg.Types)
	}

	return nil
}
