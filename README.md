# gateway

# Deploy
## model server
模型服务需要支持openai接口，可以使用fastchat或api-for-open-llm等库启动。
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

```
bs_model:
  self-driving-v1: ['http://127.0.0.1:8089/api/v1']
```



启动：

```./gateway start --config ./config.yml```