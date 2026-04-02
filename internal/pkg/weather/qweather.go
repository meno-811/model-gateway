package weather

import (
	"context"
	"fmt"
	"morning/internal/pkg/conf"
	"morning/internal/pkg/logger"
	"sync"

	"github.com/go-resty/resty/v2"
)

var (
	globalQWeather *QWeather
	cfg            = conf.GetConfig()
	loadOnce       sync.Once
)

type QWeather struct {
	MyAPIHost  string
	ApiKeys    []string
	httpClient *resty.Client
}

func NewQWeather() *QWeather {
	loadOnce.Do(func() {
		globalQWeather = &QWeather{
			MyAPIHost:  cfg.QWeather.APIHost,
			ApiKeys:    cfg.QWeather.APIKeys,
			httpClient: resty.New(),
		}
	})
	return globalQWeather
}

func GetQWeather() *QWeather {
	if globalQWeather == nil {
		NewQWeather()
	}
	return globalQWeather
}

func (q *QWeather) GetWeatherNow(ctx context.Context, cityID string) (QWeatherNowResp, error) {
	var resp QWeatherNowResp

	req := q.httpClient.R().
		SetQueryParams(map[string]string{
			"location": cityID,
			"key":      q.ApiKeys[0],
		}).
		SetResult(&resp)

	response, err := req.Get(q.MyAPIHost + "/weather/now")
	if err != nil {
		logger.Errorf("查询天气失败：%v", err)
		return QWeatherNowResp{}, err
	}

	if response.IsError() {
		logger.Errorf("查询天气出现错误：%v", response.StatusCode())
		return QWeatherNowResp{}, fmt.Errorf("查询天气出现错误：%d", response.StatusCode())
	}

	return resp, nil
}

func (q *QWeather) GetWeatherToday(ctx context.Context, cityID string) (QWeatherTodayResp, error) {
	var resp QWeatherTodayResp

	req := q.httpClient.R().
		SetQueryParams(map[string]string{
			"location": cityID,
			"key":      q.ApiKeys[0],
		}).
		SetResult(&resp)

	response, err := req.Get(q.MyAPIHost + "/weather/24h")
	if err != nil {
		logger.Errorf("查询天气失败：%v", err)
		return QWeatherTodayResp{}, err
	}

	if response.IsError() {
		logger.Errorf("查询天气出现错误：%v", response.StatusCode())
		return QWeatherTodayResp{}, fmt.Errorf("查询天气出现错误：%d", response.StatusCode())
	}

	return resp, nil
}
