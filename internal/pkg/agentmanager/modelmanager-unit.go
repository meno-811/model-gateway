package agentmanager

import (
	"context"
	"morning/internal/pkg/logger"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

func generateWithOpenAI(ctx context.Context, in []*schema.Message, APIKey, baseURL, modelname string, opts ...model.Option) (outMsg *schema.Message, err error) {
	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		Model:   modelname,
		APIKey:  APIKey,
		BaseURL: baseURL,
	})
	if err != nil {
		logger.Errorf("创建openai chat model失败: %v", err)
		return nil, err
	}
	outMsg, err = chatModel.Generate(ctx, in, opts...)
	if err != nil {
		logger.Errorf("openai chat model 生成失败: %v", err)
		return nil, err
	}
	return outMsg, nil
}

func generateWithDeepseek(ctx context.Context, in []*schema.Message, APIKey, baseURL, modelname string, opts ...model.Option) (outMsg *schema.Message, err error) {
	chatModel, err := deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
		Model:   modelname,
		APIKey:  APIKey,
		BaseURL: baseURL,
	})
	if err != nil {
		logger.Errorf("创建deepseek chat model失败: %v", err)
		return nil, err
	}
	outMsg, err = chatModel.Generate(ctx, in, opts...)
	if err != nil {
		logger.Errorf("deepseek chat model 生成失败: %v", err)
		return nil, err
	}
	return outMsg, nil
}

// IsAuthError 判断是否为认证错误
func IsAuthError(err error) bool {
	if err == nil {
		return false
	}

	// 检查是否包含认证相关关键词
	errMsg := strings.ToLower(err.Error())
	authKeywords := []string{
		"401",
		"api key",
		"apikey",
		"authentication",
		"unauthorized",
		"invalid key",
	}

	for _, keyword := range authKeywords {
		if strings.Contains(errMsg, keyword) {
			return true
		}
	}
	return false
}

// IsBillingError 判断是否为计费错误
func IsBillingError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := strings.ToLower(err.Error())
	billingKeywords := []string{
		"payment required",
		"billing",
		"quota exceeded",
		"balance insufficient",
		"account suspended",
		"out of credit",
		"billing overdue",
		"insufficient funds",
	}

	for _, keyword := range billingKeywords {
		if strings.Contains(errMsg, keyword) {
			return true
		}
	}
	return false
}
