之前的一个cron工具，整体是用于定时推送消息的（现在看来很多代码都写得有点弱智）
不过在/internal/pkg/agentmanager/modelmanager.go里憋了一个功能函数，把免费模型都塞进了一个队列里来提供成一个整体的模型调用，且加了错误处理
觉得可以拓展一些写成一个单独的服务，然后就是赛博捡垃圾丢进去一起炼化

写这个项目的时候觉得一个agent就是一个系统提示词，然后ModelUseImpl来选择这个agent应该是用付费模型还是免费模型，大概就等于一个agent聪明程度了
## 中转服务 (cmd/gateway)

把免费 API Key 组成池，轮询调用，并对外提供 OpenAI / Anthropic 格式的接口：

- `POST /v1/chat/completions`（OpenAI 格式）、`POST /v1/messages`（Anthropic 格式）、`GET /v1/models`
- 上游 Key 的协议与调用方协议不同时会自动互转（文本对话，支持流式）；相同时直接透传
- 失败自动换下一个 Key；401/402/403 立即禁用，连续失败 3 次禁用
- 管理页面 `http://host:8080/admin`：运行时添加 / 删除 / 启用 / 禁用 Key（Basic Auth，密码为 `ADMIN_TOKEN`），持久化到 `KEYS_FILE`

```
ADMIN_TOKEN=xxx GATEWAY_API_KEY=yyy go run ./cmd/gateway
```

环境变量：`ADMIN_TOKEN`（必填，未设置则管理页禁用）、`GATEWAY_API_KEY`（调用方凭证，空则不校验）、`LISTEN_ADDR`（默认 `:8080`）、`KEYS_FILE`（默认 `keys.json`）。
添加 Key 时 Base URL 需包含版本前缀，如 `https://api.openai.com/v1`、`https://api.anthropic.com/v1`。
