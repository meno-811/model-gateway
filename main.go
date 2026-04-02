package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"morning/internal/pkg/agentmanager"
	"morning/internal/pkg/conf"
	datainfo "morning/internal/pkg/dataInfo"
	"morning/internal/pkg/logger"
	"morning/internal/pkg/napcat"
	"morning/internal/pkg/weather"
)

func main() {
	cfg := conf.GetConfig()
	napcatClient := napcat.GetnapcatClient()
	qweather := weather.GetQWeather()
	ctx := context.Background()

	ag := agentmanager.BuildAgent(cfg.QWeather.AgentPrompt, agentmanager.ModelUseFree)

	var weatherInfo weather.QWeatherTodayResp
	var err error
	maxRetries := 3

	for i := 0; i < maxRetries; i++ {
		weatherInfo, err = qweather.GetWeatherToday(ctx, "101280601")
		if err == nil {
			break
		}

		logger.Errorf("获取天气信息失败，第 %d 次重试: %v", i+1, err.Error())

		if i < maxRetries-1 {
			time.Sleep(time.Second * 2)
		}
	}

	if err != nil {
		logger.Errorf("获取天气信息失败，已重试 %d 次: %v", maxRetries, err.Error())
		return
	}

	holidayInfo, err := datainfo.GetHolidayInfo(ctx, "")
	if err != nil {
		logger.Errorf("获取节假日信息失败: %v", err.Error())
		return
	}

	dataMap := map[string]interface{}{
		"weather": weatherInfo,
		"holiday": holidayInfo,
	}

	dataJson, err := json.Marshal(dataMap)
	if err != nil {
		logger.Errorf("数据转换为 JSON 失败: %v", err.Error())
		return
	}

	msgToUser, err := ag.Generate(ctx, ag.ModelUseImpl, string(dataJson))
	if err != nil {
		logger.Errorf("生成消息失败: %v", err.Error())
		return
	}
	fmt.Printf("msgToUser: %s\n", msgToUser)
	for _, userID := range cfg.QWeather.Subscribers {
		napcatClient.SentPrivateMessage(ctx, userID, msgToUser)
	}
}
