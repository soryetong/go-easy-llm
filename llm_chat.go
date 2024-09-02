package easyllm

import (
	"context"
	"fmt"
	"github.com/soryetong/go-easy-llm/easyai"
	"os"
)

type ChatClient struct {
	*easyai.ClientConfig
	LLMChatInterface
}

type LLMChatInterface interface {
	SetCustomParams(params interface{})

	NormalChat(ctx context.Context, request *easyai.ChatRequest) (*easyai.ChatResponse, interface{}, error)
	StreamChat(ctx context.Context, request *easyai.ChatRequest) (<-chan *easyai.ChatResponse, error)
}

func NewChatClient(config *easyai.ClientConfig) *ChatClient {
	return &ChatClient{
		config,
		getLLM(config),
	}
}

func (c *ChatClient) SetGlobalParams(params interface{}) *ChatClient {
	c.SetCustomParams(params)

	return c
}

func getLLM(cfg *easyai.ClientConfig) LLMChatInterface {
	switch cfg.Types {
	case easyai.ChatTypeQWen:
		return &easyai.QWenChat{Config: cfg}
	case easyai.ChatTypeHunYuan:
		if cfg.SecretId == "" || cfg.SecretKey == "" {
			_, _ = fmt.Fprintf(os.Stderr, "\n\n [go-easy-llm] \n"+
				"  获取Client异常: 请配置SecretId和SecretKey,{ %s } \n\n", cfg.Types)

			os.Exit(-1)
		}
		return &easyai.HunYuanChat{Config: cfg}
	default:
		_, _ = fmt.Fprintf(os.Stderr, "\n\n [go-easy-llm] \n"+
			"  获取Client异常: 无效的LLM配置,{ %s } \n\n", cfg.Types)

		os.Exit(-1)
	}

	return nil
}
