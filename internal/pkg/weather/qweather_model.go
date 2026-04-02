package weather

// 查询当时天气返回体
type QWeatherNowResp struct {
	Code       string `json:"code"`       // 状态码，200 表示成功
	UpdateTime string `json:"updateTime"` // 接口更新时间，ISO-8601 格式
	FxLink     string `json:"fxLink"`     // 天气主页链接

	Now Now `json:"now"` // 实时观测数据

	Refer Refer `json:"refer"` // 数据来源与授权信息
}

// Now 字段内部结构
type Now struct {
	ObsTime   string `json:"obsTime"`   // 观测时间
	Temp      string `json:"temp"`      // 温度，℃
	FeelsLike string `json:"feelsLike"` // 体感温度，℃
	Icon      string `json:"icon"`      // 天气图标代码
	Text      string `json:"text"`      // 天气文字描述
	Wind360   string `json:"wind360"`   // 风向，角度
	WindDir   string `json:"windDir"`   // 风向，文字
	WindScale string `json:"windScale"` // 风力等级
	WindSpeed string `json:"windSpeed"` // 风速，km/h
	Humidity  string `json:"humidity"`  // 相对湿度，%
	Precip    string `json:"precip"`    // 降水量，mm
	Pressure  string `json:"pressure"`  // 大气压，hPa
	Vis       string `json:"vis"`       // 能见度，km
	Cloud     string `json:"cloud"`     // 云量，%
	Dew       string `json:"dew"`       // 露点温度，℃
}

// Refer 字段内部结构
type Refer struct {
	Sources []string `json:"sources"` // 原始数据源列表
	License []string `json:"license"` // 授权说明
}

// 查询当天天气返回体
type QWeatherTodayResp struct {
	Code       string    `json:"code"`
	UpdateTime string    `json:"updateTime"`
	FxLink     string    `json:"fxLink"`
	Hourly     []Hourly  `json:"hourly"`
	Refer      ReferInfo `json:"refer"`
}

// 逐小时条目
type Hourly struct {
	FxTime    string `json:"fxTime"`    // 预报时间，ISO8601
	Temp      string `json:"temp"`      // 温度，单位 ℃
	Icon      string `json:"icon"`      // 天气图标代码
	Text      string `json:"text"`      // 天气文字描述
	Wind360   string `json:"wind360"`   // 风向，角度
	WindDir   string `json:"windDir"`   // 风向，文字
	WindScale string `json:"windScale"` // 风力等级
	WindSpeed string `json:"windSpeed"` // 风速，km/h
	Humidity  string `json:"humidity"`  // 相对湿度，%
	Pop       string `json:"pop"`       // 降水概率，%
	Precip    string `json:"precip"`    // 降水量，mm
	Pressure  string `json:"pressure"`  // 大气压，hPa
	Cloud     string `json:"cloud"`     // 云量，%
	Dew       string `json:"dew"`       // 露点温度，℃
}

// 数据来源说明
type ReferInfo struct {
	Sources []string `json:"sources"`
	License []string `json:"license"`
}
