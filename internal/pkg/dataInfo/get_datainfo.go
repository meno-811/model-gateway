package datainfo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type HolidayResponse struct {
	Code    int         `json:"code"`
	Type    TypeInfo    `json:"type"`
	Holiday HolidayInfo `json:"holiday"`
}

type TypeInfo struct {
	Type int    `json:"type"`
	Name string `json:"name"`
	Week int    `json:"week"`
}

type HolidayInfo struct {
	Holiday bool   `json:"holiday"`
	Name    string `json:"name"`
	Wage    int    `json:"wage"`
	Date    string `json:"date"`
	Rest    int    `json:"rest"`
}

func GetHolidayInfo(ctx context.Context, dateStr string) (*HolidayResponse, error) {
	if dateStr == "" {
		currentTime := time.Now()
		dateStr = currentTime.Format("2006-01-02")
	}

	apiURL := fmt.Sprintf("http://timor.tech/api/holiday/info/%s", dateStr)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API返回错误状态码: %d", resp.StatusCode)
	}

	var holidayResp HolidayResponse
	if err := json.NewDecoder(resp.Body).Decode(&holidayResp); err != nil {
		return nil, fmt.Errorf("解析JSON失败: %w", err)
	}

	return &holidayResp, nil
}
