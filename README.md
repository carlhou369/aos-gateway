# gateway

# Deploy
## model worker
worker目前需支持openai接口，可以使用fastchat或api-for-open-llm等库启动。
参考：
1. https://github.com/xusenlinzy/api-for-open-llm/blob/master/docs/SCRIPT.md
2. https://github.com/lm-sys/FastChat/blob/main/docs/openai_api.md

## gateway
### 配置
参考localconfig.yml，新建配置文件config.yml

```cp ./localconfig.yml ./config.yml```

#### openai_key
配置openai token，请求模型为"gpt"时使用此token代理请求openai接口。

#### bs_model
bs_model为支持的模型和模型worker的url。例如下面配置中模型名称为“self-driving-v1”，一个worker的url为'http://127.0.0.1:8089/api/v1'

#### max_pending
限流，最大等待中的api请求，超过返回失败。

```
bs_model:
  self-driving-v1: ['http://127.0.0.1:8089/api/v1']
```



### 启动

启动monogo：

```docker compose up -d```

启动gateway：

```./gateway start --config ./config.yml```

# API

## 模型对话
**/api/question**

message_id唯一标识一轮对话，conversation_id唯一标识连续对话。message为llm请求问题。

使用cookie可以自动连续对话，配置conversation_id可以切换对话，message为“clear”清空对话。

请求：

```
{
    "message":"hello",
    "message_id":"",
    "conversation_id":"",
    "model":"vicuna-7b-v1.5"
}
```

返回：

```
{
    "ret": 200,
    "msg": "",
    "data": "{\"text\":\"Hello! How can I help you today? Is there something you would like to talk about or ask me a question? I'm here to assist you with any information or tasks you might need help with.\",\"messageId\":\"d35fb408-14e8-484d-9c50-5a69505ee89b\",\"conversationId\":\"ff2583db-2ab5-434a-8993-de28947ae63a\",\"model\":\"vicuna-7b-v1.5\"}"
}
```

## 模型worker加入gateway

**/api/register**

请求：

```
{
    "model":"vicuna-7b-v1.5",
    "url":"http://localhost:8443/v1/chat/completions"
}
```

返回：

```
{
    "ret": 200,
    "msg": "ok",
    "data": ""
}
```