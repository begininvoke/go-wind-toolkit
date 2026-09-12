package ai

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ConfigPath 返回 AI 配置的持久化文件路径:
// <UserConfigDir>/go-wind-toolkit/ai-config.json
// (Windows: %APPDATA%\go-wind-toolkit\ai-config.json)
// GUI 与 CLI 共用,配置一次两边生效;环境变量 GOWIND_AI_* 仍可覆盖。
func ConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(dir, "go-wind-toolkit", "ai-config.json"), nil
}

// LoadConfig 读取持久化的 AI 配置;文件缺失、损坏或字段不全时
// 回落到默认配置,保证调用方永远拿到可用值。
func LoadConfig() *Config {
	cfg := DefaultConfig()

	path, err := ConfigPath()
	if err != nil {
		return cfg
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg
	}

	var saved Config
	if err := json.Unmarshal(data, &saved); err != nil {
		return cfg
	}

	// 逐字段合并:持久化文件里缺省的字段(如升级新增)保持默认值。
	if saved.Provider != "" {
		cfg.Provider = saved.Provider
	}
	if saved.BaseURL != "" {
		cfg.BaseURL = saved.BaseURL
	}
	if saved.APIKey != "" {
		cfg.APIKey = saved.APIKey
	}
	if saved.AzureAPIVersion != "" {
		cfg.AzureAPIVersion = saved.AzureAPIVersion
	}
	if saved.Model != "" {
		cfg.Model = saved.Model
	}
	// Temperature=0 是合法取值(确定性输出),不能当缺省回退到默认值;
	// 仅排除负数与超出 [0,2] 的脏数据。
	if saved.Temperature >= 0 && saved.Temperature <= 2 {
		cfg.Temperature = saved.Temperature
	}
	if saved.MaxTokens > 0 {
		cfg.MaxTokens = saved.MaxTokens
	}
	return cfg
}

// SaveConfig 将 AI 配置持久化到磁盘(最佳努力,失败由调用方决定是否提示)。
func SaveConfig(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("AI 配置为空")
	}
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	// 0600:文件内含 API 密钥,仅限当前用户读取。
	return os.WriteFile(path, data, 0o600)
}
