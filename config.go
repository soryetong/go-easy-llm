package easyllm

import (
	"github.com/soryetong/go-easy-llm/easyai"
	"net/http"
	"net/url"
)

func DefaultConfig(token string, types easyai.LLMType) *easyai.ClientConfig {
	return &easyai.ClientConfig{
		Types:      types,
		Token:      token,
		HttpClient: &http.Client{},
	}
}

func DefaultConfigWithProxy(token string, types easyai.LLMType, proxyUrl string) *easyai.ClientConfig {
	proxy, _ := url.Parse(proxyUrl)
	httpClient := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxy),
		},
	}

	return &easyai.ClientConfig{
		Types:      types,
		Token:      token,
		HttpClient: httpClient,
	}
}

func DefaultConfigWithSecret(secretId, secretKey string, types easyai.LLMType) *easyai.ClientConfig {
	return &easyai.ClientConfig{
		Types:      types,
		SecretId:   secretId,
		SecretKey:  secretKey,
		HttpClient: &http.Client{},
	}
}

func DefaultConfigWithSecretAndProxy(secretId, secretKey, proxyUrl string, types easyai.LLMType) *easyai.ClientConfig {
	proxy, _ := url.Parse(proxyUrl)
	httpClient := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxy),
		},
	}

	return &easyai.ClientConfig{
		Types:      types,
		SecretId:   secretId,
		SecretKey:  secretKey,
		HttpClient: httpClient,
	}
}
