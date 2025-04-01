package easyllm

import (
	"net/http"
	"net/url"

	"github.com/soryetong/go-easy-llm/easyai/chatmodule"
)

func DefaultConfig(token string, types chatmodule.LLMType) *chatmodule.ClientConfig {
	return &chatmodule.ClientConfig{
		Types:      types,
		Token:      token,
		HttpClient: &http.Client{},
	}
}

func DefaultConfigWithProxy(token string, types chatmodule.LLMType, proxyUrl string) *chatmodule.ClientConfig {
	httpClient := &http.Client{}
	if proxyUrl != "" {
		proxy, _ := url.Parse(proxyUrl)
		httpClient.Transport = &http.Transport{
			Proxy: http.ProxyURL(proxy),
		}
	}

	return &chatmodule.ClientConfig{
		Types:      types,
		Token:      token,
		HttpClient: httpClient,
	}
}

func DefaultConfigWithSecret(secretId, secretKey string, types chatmodule.LLMType) *chatmodule.ClientConfig {
	return &chatmodule.ClientConfig{
		Types:      types,
		SecretId:   secretId,
		SecretKey:  secretKey,
		HttpClient: &http.Client{},
	}
}

func DefaultConfigWithSecretAndProxy(secretId, secretKey string, types chatmodule.LLMType, proxyUrl string) *chatmodule.ClientConfig {
	httpClient := &http.Client{}
	if proxyUrl != "" {
		proxy, _ := url.Parse(proxyUrl)
		httpClient.Transport = &http.Transport{
			Proxy: http.ProxyURL(proxy),
		}
	}

	return &chatmodule.ClientConfig{
		Types:      types,
		SecretId:   secretId,
		SecretKey:  secretKey,
		HttpClient: httpClient,
	}
}
