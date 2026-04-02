package conf

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"

	"morning/internal/pkg/logger"
)

// Config 配置结构体示例
// 添加新配置项只需在此处添加字段即可
type Config struct {
	Napcat       NapcatConfig       `yaml:"napcat"`
	QWeather     QWeatherConfig     `yaml:"qweather"`
	LLMSuppliers LLMSuppliersConfig `yaml:"llmsuppliers"`
}

type QWeatherConfig struct {
	AgentPrompt string   `yaml:"agentPrompt"`
	APIHost     string   `yaml:"apiHost"`
	APIKeys     []string `yaml:"apiKey"`
	Subscribers []string `yaml:"subscribers"`
}

type LLMSuppliersConfig struct {
	TrialSuppliers map[string]ModelConfig `yaml:"trial_suppliers"`
	PardSuppliers  map[string]ModelConfig `yaml:"pard_suppliers"`
}

type ModelConfig struct {
	BaseURL      string   `yaml:"baseURL"`
	ProtocolType string   `yaml:"protocol_type"`
	APIKeys      []string `yaml:"api_keys"`
	Models       []string `yaml:"models"`
}

type NapcatConfig struct {
	BaseURL  string `yaml:"baseURL"`
	Token    string `yaml:"token"`
	AutherID string `yaml:"autherID"`
}

var (
	globalConfig *Config
	loadOnce     sync.Once
)

func getConfigPath() string {
	// 获取二进制文件绝对路径
	execPath, err := os.Executable()
	if err != nil {
		logger.Errorf("获取可执行文件路径失败: %w", err)
		panic(err)
	}

	// 获取二进制文件所在目录
	execDir := filepath.Dir(execPath)
	configPath := filepath.Join(execDir, "config.yaml")

	// 如果可执行文件目录中不存在配置文件，尝试当前工作目录
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		workDir, err := os.Getwd()
		if err == nil {
			workConfigPath := filepath.Join(workDir, "config.yaml")
			if _, err := os.Stat(workConfigPath); err == nil {
				return workConfigPath
			}
		}
	}

	return configPath
}

// Load 加载配置，线程安全，仅第一次调用会实际加载
func load() (*Config, error) {
	loadOnce.Do(func() {
		// 1. 加载 YAML 配置文件
		configPath := getConfigPath()
		data, err := os.ReadFile(configPath)
		if err != nil {
			if !os.IsNotExist(err) {
				logger.Errorf("读取 %s 文件失败: %w", configPath, err)
				panic(err)
			}
			// 文件不存在时使用默认配置
			logger.Warnf("警告: %s 不存在，使用默认配置", configPath)
			globalConfig = &Config{}
			return
		}

		// 2. 解析 YAML
		cfg := &Config{}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			logger.Errorf("解析 YAML 失败: %w", err)
			panic(err)
		}

		// 3. 打印加载的配置（自动屏蔽敏感字段）
		printLoadedConfig(cfg)
		globalConfig = cfg
	})

	return globalConfig, nil
}

// GetConfig 获取已加载的配置
// 如果配置未加载，会 panic
func GetConfig() *Config {
	if globalConfig == nil {
		var err error
		globalConfig, err = load()
		if err != nil {
			logger.Errorf("加载配置失败: %w", err)
			panic(err)
		}
	}
	return globalConfig
}

// printLoadedConfig 递归打印加载的配置信息，自动屏蔽敏感字段
func printLoadedConfig(cfg interface{}, indent ...int) {
	currentIndent := 0
	if len(indent) > 0 {
		currentIndent = indent[0]
	}

	prefix := strings.Repeat("  ", currentIndent)

	if currentIndent == 0 {
		logger.Infof("配置加载完成")
		fmt.Println("==================================================")
	}

	v := reflect.ValueOf(cfg)
	t := v.Type()

	// 处理指针
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
		t = v.Type()
	}

	// 只处理结构体
	if v.Kind() != reflect.Struct {
		logger.Infof("%s%v", prefix, v.Interface())
		return
	}

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)
		yamlTag := fieldType.Tag.Get("yaml")

		// 解析 yaml 标签，提取配置名称
		parts := strings.Split(yamlTag, ",")
		configName := parts[0]
		if configName == "" || configName == "-" {
			continue
		}

		// // 检查是否是敏感字段
		// isSensitive := fieldType.Tag.Get("sensitive") == "true" ||
		// 	strings.Contains(strings.ToLower(fieldType.Name), "password") ||
		// 	strings.Contains(strings.ToLower(fieldType.Name), "secret") ||
		// 	strings.Contains(strings.ToLower(fieldType.Name), "key")

		value := field.Interface()

		// 显示来源（从文件加载或默认值）
		source := "配置文件"
		if reflect.DeepEqual(value, reflect.Zero(field.Type()).Interface()) {
			source = "默认值"
		}

		// if isSensitive {
		// 	value = "******" // 屏蔽敏感信息
		// }

		fmt.Printf("%s%-20s = %-40v [%s]\n", prefix, configName, value, source)

		// 递归处理嵌套结构
		if field.Kind() == reflect.Struct || (field.Kind() == reflect.Ptr && field.Elem().Kind() == reflect.Struct) {
			printLoadedConfig(value, currentIndent+1)
		}

		// 处理切片和 map
		if field.Kind() == reflect.Slice && field.Len() > 0 {
			fmt.Printf("%s%s:\n", prefix, configName)
			for j := 0; j < field.Len(); j++ {
				fmt.Printf("%s  - %v\n", prefix, field.Index(j).Interface())
			}
		}

		if field.Kind() == reflect.Map && field.Len() > 0 {
			fmt.Printf("%s%s:\n", prefix, configName)
			for _, key := range field.MapKeys() {
				val := field.MapIndex(key)
				fmt.Printf("%s  %s: %v\n", prefix, key.Interface(), val.Interface())
			}
		}
	}

	if currentIndent == 0 {
		fmt.Println("==================================================")
	}
}
