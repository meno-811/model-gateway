package agentmanager

import (
	"context"

	"github.com/cloudwego/eino/schema"
	// 根据你使用的模型选择对应的包
)

type AgentManager struct {
	SystemPrompt string
	ModelUseImpl ModelUseImpl
}
type ModelUseImpl string

const (
	ModelUseFree ModelUseImpl = "free" // 免费模型
	ModelUsePard ModelUseImpl = "pard" // 付费模型
)

func BuildAgent(systemPrompt string, modelUseImpl ModelUseImpl) AgentManager {
	return AgentManager{
		SystemPrompt: systemPrompt,
		ModelUseImpl: modelUseImpl,
	}
}

// 这个useModel可为空，为空时使用默认模型
func (a *AgentManager) Generate(ctx context.Context, modelUseImpl ModelUseImpl, userPrompt string) (string, error) {
	if modelUseImpl == "" {
		modelUseImpl = a.ModelUseImpl
	}
	template := []*schema.Message{
		schema.SystemMessage(a.SystemPrompt),
		schema.UserMessage(userPrompt),
	}
	model := GetModelManager()
	return model.Generate(ctx, modelUseImpl, template)
}
