package agentmanager

import (
	"context"
	"errors"
	"morning/internal/pkg/conf"
	"morning/internal/pkg/logger"
	"morning/internal/pkg/napcat"
	"sync"

	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

var (
	cfg                = conf.GetConfig()
	napcatClient       = napcat.GetnapcatClient()
	globalModelManager *ModelManager
	loadOnce           sync.Once
)

type ModelUse interface {
	GenerateWithFree(ctx context.Context, model string, prompt string) (string, error) // 免费模型生成
	GenerateWithPard(ctx context.Context, model string, prompt string) (string, error) // 付费模型生成
	BuildAgent(ctx context.Context, model string, prompt string) (string, error)       // 构建智能体
	CheckModel(ctx context.Context)                                                    // 检查配置的模型都是否可用，日志会上报给管理员
}

// ======================================================================================

type APIkeys struct {
	errorStreak  int
	apiKeys      string
	supplier     string
	model        []string
	protocolType string
	baseURL      string
}

type ModelManager struct {
	TrialAPIKeys chan APIkeys // 免费模型API
	PardAPIKeys  chan APIkeys // 付费模型API
}

func InitModelManager() *ModelManager {
	loadOnce.Do(func() {
		globalModelManager = &ModelManager{
			TrialAPIKeys: make(chan APIkeys, 30),
			PardAPIKeys:  make(chan APIkeys, 30),
		}
		for suppliername, modelConfig := range cfg.LLMSuppliers.TrialSuppliers {
			for _, apiKey := range modelConfig.APIKeys {
				globalModelManager.TrialAPIKeys <- APIkeys{
					errorStreak:  0,
					apiKeys:      apiKey,
					supplier:     suppliername,
					model:        modelConfig.Models,
					protocolType: modelConfig.ProtocolType,
					baseURL:      modelConfig.BaseURL,
				}
			}
		}
		for suppliername, modelConfig := range cfg.LLMSuppliers.PardSuppliers {
			for _, apiKey := range modelConfig.APIKeys {
				globalModelManager.PardAPIKeys <- APIkeys{
					errorStreak:  0,
					apiKeys:      apiKey,
					supplier:     suppliername,
					model:        modelConfig.Models,
					protocolType: modelConfig.ProtocolType,
					baseURL:      modelConfig.BaseURL,
				}
			}
		}
	})
	return globalModelManager
}

func GetModelManager() *ModelManager {
	return InitModelManager()
}

func (m *ModelManager) Generate(ctx context.Context, modelUse ModelUseImpl, prompt []*schema.Message) (string, error) {
	if modelUse == ModelUseFree {
		return m.GenerateWithFree(ctx, prompt)
	}
	return m.GenerateWithPard(ctx, prompt)
}

// GenerateWithFree 自动换 Key 重试版
func (m *ModelManager) GenerateWithFree(ctx context.Context, prompt []*schema.Message) (string, error) {
	maxTries := cap(m.TrialAPIKeys)
	for attempt := 0; attempt < maxTries; attempt++ {
		select {
		case apiKey := <-m.TrialAPIKeys:
			// ---------- 1. 建模型 ----------
			var chatModel model.ToolCallingChatModel
			var err error
			switch apiKey.protocolType {
			case "openai":
				chatModel, err = openai.NewChatModel(ctx, &openai.ChatModelConfig{
					Model:   apiKey.model[0],
					APIKey:  apiKey.apiKeys,
					BaseURL: apiKey.baseURL,
				})
			case "deepseek":
				chatModel, err = deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
					Model:   apiKey.model[0],
					APIKey:  apiKey.apiKeys,
					BaseURL: apiKey.baseURL,
				})
			default:
				logger.Errorf("unknown protocol type: %s", apiKey.protocolType)
				// 这个 Key 配置本身有问题，直接丢弃，继续下一轮
				continue
			}
			if err != nil {
				logger.Errorf("create chat model failed: %v", err)
				// 建模型失败，同样丢弃该 Key
				continue
			}

			// ---------- 2. 调 Generate ----------
			outMsg, err := chatModel.Generate(ctx, prompt)
			if err != nil {
				if IsAuthError(err) {
					logger.Errorf("API Key 无效或已过期，已被禁用: %v", err)
					napcatClient.Sendlog(ctx, "API Key 无效或已过期，已被禁用: "+err.Error())
					// 认证类错误直接丢弃
					continue
				}
				// 其他错误，累计错误次数
				apiKey.errorStreak++
				if apiKey.errorStreak >= 3 {
					logger.Errorf("API Key 连续错误超过3次，已被禁用: %v", err)
					napcatClient.Sendlog(ctx, "API Key 连续错误超过3次，已被禁用: "+err.Error())
					// 超过阈值，丢弃
					continue
				}
				// 未达阈值，把 Key 还回去，继续下一轮
				m.TrialAPIKeys <- apiKey
				continue
			}

			// ---------- 3. 成功 ----------
			apiKey.errorStreak = 0
			m.TrialAPIKeys <- apiKey
			return outMsg.Content, nil

		case <-ctx.Done():
			logger.Errorf("context deadline exceeded: %v", ctx.Err())
			return "", ctx.Err()
		}
	}
	return "", errors.New("all trial API keys exhausted")
}

func (m *ModelManager) GenerateWithPard(ctx context.Context, prompt []*schema.Message) (string, error) {
	maxTries := cap(m.PardAPIKeys)
	for attempt := 0; attempt < maxTries; attempt++ {
		select {
		case apiKey := <-m.PardAPIKeys:
			// ---------- 1. 建模型 ----------
			var chatModel model.ToolCallingChatModel
			var err error
			switch apiKey.protocolType {
			case "openai":
				chatModel, err = openai.NewChatModel(ctx, &openai.ChatModelConfig{
					Model:   apiKey.model[0],
					APIKey:  apiKey.apiKeys,
					BaseURL: apiKey.baseURL,
				})
			case "deepseek":
				chatModel, err = deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
					Model:   apiKey.model[0],
					APIKey:  apiKey.apiKeys,
					BaseURL: apiKey.baseURL,
				})
			default:
				logger.Errorf("unknown protocol type: %s", apiKey.protocolType)
				// 这个 Key 配置本身有问题，直接丢弃，继续下一轮
				continue
			}
			if err != nil {
				logger.Errorf("create chat model failed: %v", err)
				// 建模型失败，同样丢弃该 Key
				continue
			}

			// ---------- 2. 调 Generate ----------
			outMsg, err := chatModel.Generate(ctx, prompt)
			if err != nil {
				if IsBillingError(err) {
					logger.Errorf("这个 API Key 没钱了！: %v", err)
					napcatClient.Sendlog(ctx, "这个 API Key 没钱了！: "+err.Error())
					// 计费类错误直接丢弃
					continue
				}
				// 其他错误，累计错误次数
				apiKey.errorStreak++
				if apiKey.errorStreak >= 3 {
					logger.Errorf("API Key 连续错误超过3次，已被禁用: %v", err)
					napcatClient.Sendlog(ctx, "API Key 连续错误超过3次，已被禁用: "+err.Error())
					// 超过阈值，丢弃
					continue
				}
				// 未达阈值，把 Key 还回去，继续下一轮
				m.TrialAPIKeys <- apiKey
				continue
			}

			// ---------- 3. 成功 ----------
			apiKey.errorStreak = 0
			m.TrialAPIKeys <- apiKey
			return outMsg.Content, nil

		case <-ctx.Done():
			logger.Errorf("context deadline exceeded: %v", ctx.Err())
			return "", ctx.Err()
		}
	}
	return "", errors.New("all trial API keys exhausted")
}
