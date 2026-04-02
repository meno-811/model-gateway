package napcat

import (
	"context"
	"fmt"
	"net/http"

	"morning/internal/pkg/conf"
	"morning/internal/pkg/logger"

	"github.com/go-resty/resty/v2"
)

var (
	globalnapcatClient *napcatClient
	cfg                = conf.GetConfig()
)

type napcatClient struct {
	baseURL    string
	token      string
	autherID   string
	httpClient *resty.Client
}

func initClient(baseURL, token, autherID string) *napcatClient {
	return &napcatClient{
		baseURL:    baseURL,
		token:      token,
		autherID:   autherID,
		httpClient: resty.New(),
	}
}

func GetnapcatClient() *napcatClient {
	if globalnapcatClient == nil {
		globalnapcatClient = initClient(cfg.Napcat.BaseURL, cfg.Napcat.Token, cfg.Napcat.AutherID)
	}
	return globalnapcatClient
}

func (c *napcatClient) SentGroupTextMessage(ctx context.Context, groupID string, textmsg string) error {
	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetHeader("Authorization", "Bearer "+c.token).
		SetHeader("Content-Type", "application/json").
		SetBody(map[string]any{
			"group_id": groupID,
			"message": Message{
				Type: "text",
				Data: map[string]string{"text": textmsg},
			},
		}).
		Post(c.baseURL + "/send_group_msg")
	if err != nil {
		logger.Errorf("发送群消息请求失败: %v", err)
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		logger.Errorf("发送群消息请求失败, 状态码: %d", resp.StatusCode())
		return fmt.Errorf("发送群消息请求失败, 状态码: %d", resp.StatusCode())
	}
	logger.Infof("发送群消息成功, 群ID: %s, 消息: %s", groupID, textmsg)
	return nil
}

func (c *napcatClient) SentPrivateMessage(ctx context.Context, userID string, textmsg string) error {
	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetHeader("Authorization", "Bearer "+c.token).
		SetHeader("Content-Type", "application/json").
		SetBody(map[string]any{
			"user_id": userID,
			"message": Message{
				Type: "text",
				Data: map[string]string{"text": textmsg},
			},
		}).
		Post(c.baseURL + "/send_private_msg")
	if err != nil {
		logger.Errorf("发送私聊消息请求失败: %v", err)
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		logger.Errorf("发送私聊消息请求失败, 状态码: %d", resp.StatusCode())
		return fmt.Errorf("发送私聊消息请求失败, 状态码: %d", resp.StatusCode())
	}
	logger.Infof("发送私聊消息成功, 用户ID: %s, 消息: %s", userID, textmsg)
	return nil
}
