<h1 align="center">go-easy-llm</h1>

<p align="center"> 又一个满足你的调用多种大模型API的轮子</p>

<p align="center">

</p>

<p align="center">

</p>

## 特点

1. 支持目前市面多家第三方大模型
2. 一套写法兼容所有平台
3. 简单配置即可灵活使用第三方 
4. 更多等你改进...

## 已支持的第三方

- [阿里云 通义千问](https://help.aliyun.com/zh/model-studio/developer-reference/tongyi-qianwen)
    
    - 自定义配置 `globalParams := new(easyai.QWenParameters)` 按需设置参数


## 当前go版本

- go 1.22

## 安装

```shell
go get github.com/soryetong/go-easy-llm
```

## 使用

1. 添加你自己的配置
```go
config := easyllm.DefaultConfig("your-token", easyai.TypeQWen)

// 第一个参数是你的token, 第二个参数是你的大模型类型
// 可用的大模型, 通过easyai.Type*** 获取
```
> 如果需要代理请求
```go
config := easyllm.DefaultConfigWithProxy("your-token", easyai.TypeQWen, "your-proxy-url")
```


2. 创建 `Chat` 客户端
```go
client := easyllm.NewChatClient(config)
```
> 创建客户端可以自定义全局配置
```go
client := easyllm.NewChatClient(config).SetGlobalParams(globalParams)

// 或
client.SetCustomParams(globalParams)

// 两个方法的最终效果都是一样的,设置一个全局的参数
```

3. 调用 `Chat` 模式大模型
> 一次性回复 `NormalChat`
```go
resp, reply, err := client.NormalChat(context.Background(), &easyai.ChatRequest{
    Model:   easyai.QWenTurboModel,
    Message: "请介绍一下自己",
})
// resp 为定义的通用类型, `easyai.ChatResponse`
// reply 为大模型返回的文本
```

> 流式回复 `StreamChat`
```go
resp, err := client.StreamChat(context.Background(), &easyai.ChatRequest{
    Model:   easyai.QWenTurboModel,
    Message: "介绍一下你自己",
})

for content := range resp {
    fmt.Println(content)
}
```

## 说明
1. `ChatRequest.Tips`：提示词，用于引导模型生成更符合要求的答案。
2. 目前只支持 `chat` 模式，绘画等功能将在后续完善


## 示例
1. 在unitest目录下有示例代码