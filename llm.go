package easyllm

import (
	"context"
	"fmt"
	"github.com/soryetong/go-easy-llm/easyai"
	"os"
)

type Client struct {
	*easyai.ClientConfig
	LLMInterface
}

type LLMInterface interface {
	SetCustomParams(params interface{})

	NormalChat(ctx context.Context, request *easyai.ChatRequest) (*easyai.ChatResponse, interface{}, error)
	StreamChat(ctx context.Context, request *easyai.ChatRequest) (<-chan *easyai.ChatResponse, error)
}

func NewClient(config *easyai.ClientConfig) *Client {
	return &Client{
		config,
		getLLM(config),
	}
}

func (c *Client) SetGlobalParams(params interface{}) *Client {
	c.SetCustomParams(params)

	return c
}

func getLLM(cfg *easyai.ClientConfig) LLMInterface {
	switch cfg.Types {
	case easyai.TypeQWen:
		return &easyai.QWenChat{Config: cfg}
	default:
		_, _ = fmt.Fprintf(os.Stderr, "\n\n [go-easy-llm] \n"+
			"  获取Client异常: 无效的LLM配置,{ %s } \n\n", cfg.Types)

		os.Exit(-1)
	}

	return nil
}
