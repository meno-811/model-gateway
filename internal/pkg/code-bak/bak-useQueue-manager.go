package codebak

import (
	"context"
	"errors"
	"morning/internal/pkg/conf"
	"morning/internal/pkg/logger"
	"morning/internal/pkg/unit"
	"sync"

	"github.com/cloudwego/eino/schema"
	// 根据你使用的模型选择对应的包
)

var (
	cfg                = conf.GetConfig()
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
	TrialAPIKeys *unit.LinkedQueue // 免费模型API
	PardAPIKeys  *unit.LinkedQueue // 付费模型API
}

func InitModelManager() *ModelManager {
	loadOnce.Do(func() {
		globalModelManager = &ModelManager{
			TrialAPIKeys: unit.NewLinkedQueue(),
			PardAPIKeys:  unit.NewLinkedQueue(),
		}
		for suppliername, modelConfig := range cfg.LLMSuppliers.TrialSuppliers {
			for _, apiKey := range modelConfig.APIKeys {
				globalModelManager.TrialAPIKeys.Enqueue(
					APIkeys{
						errorStreak:  0,
						apiKeys:      apiKey,
						supplier:     suppliername,
						model:        modelConfig.Models,
						protocolType: modelConfig.ProtocolType,
						baseURL:      modelConfig.BaseURL,
					},
				)
			}
		}
		for suppliername, modelConfig := range cfg.LLMSuppliers.PardSuppliers {
			for _, apiKey := range modelConfig.APIKeys {
				globalModelManager.PardAPIKeys.Enqueue(
					APIkeys{
						errorStreak:  0,
						apiKeys:      apiKey,
						supplier:     suppliername,
						model:        modelConfig.Models,
						protocolType: modelConfig.ProtocolType,
						baseURL:      modelConfig.BaseURL,
					},
				)
			}
		}
	})
	return globalModelManager
}

func GetModelManager() *ModelManager {
	return InitModelManager()
}

type ModelUseImpl string

const (
	ModelUseFree ModelUseImpl = "free" // 免费模型
	ModelUsePard ModelUseImpl = "pard" // 付费模型
)

func (m *ModelManager) Generate(ctx context.Context, modelUse ModelUseImpl, prompt []*schema.Message) (string, error) {
	if modelUse == ModelUseFree {
		return m.GenerateWithFree(ctx, prompt)
	}
	return m.GenerateWithPard(ctx, prompt)
}

func (m *ModelManager) GenerateWithFree(ctx context.Context, prompt []*schema.Message) (string, error) {
	apiKey, ok := m.TrialAPIKeys.Dequeue().(APIkeys)
	if !ok {
		logger.Error("no free api key")
		return "", errors.New("no free api key")
	}
	// switch apiKey.protocolType {
	// case "openai":
	// 	outMsg, err := generateWithOpenAI(ctx, prompt, apiKey.apiKeys, apiKey.baseURL, apiKey.model[0])
	// 	if err != nil {
	// 		logger.Errorf("openai chat model 生成失败: %v", err)
	// 		return "", err
	// 	}
	// 	return outMsg.Content, nil
	// case "deepseek":
	// 	outMsg, err := generateWithDeepseek(ctx, prompt, apiKey.apiKeys, apiKey.baseURL, apiKey.model[0])
	// 	if err != nil {
	// 		logger.Errorf("deepseek chat model 生成失败: %v", err)
	// 		return "", err
	// 	}
	// 	return outMsg.Content, nil
	// default:
	// 	logger.Errorf("unknown protocol type: %s", apiKey.protocolType)
	// 	return "", errors.New("unknown protocol type")
	// }
	apiKey.errorStreak++
	m.TrialAPIKeys.Enqueue(apiKey)
	return "", errors.New("api key error")
}

func (m *ModelManager) GenerateWithPard(ctx context.Context, prompt []*schema.Message) (string, error) {
	return "", nil
}
