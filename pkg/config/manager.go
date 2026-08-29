package config

import (
	"Q-Solver/pkg/logger"
	"Q-Solver/pkg/shortcut"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

type ConfigManager struct {
	config      Config
	mu          sync.RWMutex
	configPath  string
	oldConfig   Config // 这是老配置
	subscribers []func(NewConfig Config, oldConfig Config)
}

func NewConfigManager() *ConfigManager {
	cm := &ConfigManager{
		config:      NewDefaultConfig(),
		oldConfig:   NewDefaultConfig(),
		subscribers: make([]func(NewConfig Config, oldConfig Config), 0),
	}
	cm.configPath = cm.getConfigPath()
	return cm
}

func (cm *ConfigManager) getConfigPath() string {
	var appDir string

	// 判断操作系统
	if runtime.GOOS == "windows" {
		appDir = filepath.Join(".", "config")
	} else {
		sysConfigDir, err := os.UserConfigDir()
		if err != nil {
			sysConfigDir = "."
		}
		appDir = filepath.Join(sysConfigDir, "Q-Solver")
	}

	if err := os.MkdirAll(appDir, 0755); err != nil {
	}
	fullPath := filepath.Join(appDir, "config.json")

	return fullPath
}

func (cm *ConfigManager) Load() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 先设置默认值
	cm.config = NewDefaultConfig()
	// 从文件加载
	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		if !os.IsNotExist(err) {
			logger.Printf("加载配置文件失败 (使用默认配置): %v", err)
		}
	} else {
		// 直接反序列化到 config 上，会覆盖默认值
		if err := json.Unmarshal(data, &cm.config); err != nil {
			logger.Printf("解析配置文件失败: %v", err)
		} else {
			// 从旧版本迁移：solve 快捷键改名为 send，新增独立 screenshot 快捷键。
			var raw struct {
				Shortcuts map[string]shortcut.KeyBinding `json:"shortcuts"`
			}
			if json.Unmarshal(data, &raw) == nil && raw.Shortcuts != nil {
				if _, hasScreenshot := raw.Shortcuts["screenshot"]; !hasScreenshot {
					if old, ok := raw.Shortcuts["solve"]; ok {
						cm.config.Shortcuts["screenshot"] = old
					}
				}
				if old, ok := raw.Shortcuts["solve"]; ok {
					cm.config.Shortcuts["send"] = shortcut.KeyBinding{ComboID: "118", KeyName: "F7"}
					_ = old
				}
				delete(cm.config.Shortcuts, "solve")
			}
		}
	}
	// 旧版本配置没有 AI 字体和窗口尺寸字段时，补齐安全默认值。
	if cm.config.AIFontSize < 10 || cm.config.AIFontSize > 32 {
		cm.config.AIFontSize = 14
	}
	// 为旧版空 Prompt 配置补充通用解题提示词。
	if cm.config.Prompt == "" {
		cm.config.Prompt = DefaultPrompt
	}
	cm.migrateLegacyRealtimeDefaults()
	cm.applyRealtimeDefaults()

	logger.Println("配置已加载")
	return nil
}

func (cm *ConfigManager) Save() error {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	data, err := json.MarshalIndent(cm.config, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	if err := os.WriteFile(cm.configPath, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	logger.Printf("配置已保存到: %s", cm.configPath)
	return nil
}

func (cm *ConfigManager) Get() Config {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.config
}

// UpdateFromJSON 增量更新配置。未出现在 JSON 中的字段保持不变，避免
// 保存截图或语音任一设置组时覆盖另一组配置。
func (cm *ConfigManager) UpdateFromJSON(jsonStr string) error {
	var patch map[string]json.RawMessage
	if err := json.Unmarshal([]byte(jsonStr), &patch); err != nil {
		return fmt.Errorf("解析配置 JSON 失败: %w", err)
	}

	cm.mu.Lock()
	baseData, err := json.Marshal(cm.config)
	if err != nil {
		cm.mu.Unlock()
		return fmt.Errorf("序列化当前配置失败: %w", err)
	}
	var merged map[string]json.RawMessage
	if err := json.Unmarshal(baseData, &merged); err != nil {
		cm.mu.Unlock()
		return fmt.Errorf("解析当前配置失败: %w", err)
	}
	for key, value := range patch {
		merged[key] = value
	}
	mergedData, err := json.Marshal(merged)
	if err != nil {
		cm.mu.Unlock()
		return fmt.Errorf("合并配置失败: %w", err)
	}
	newConfig := NewDefaultConfig()
	if err := json.Unmarshal(mergedData, &newConfig); err != nil {
		cm.mu.Unlock()
		return fmt.Errorf("解析合并配置失败: %w", err)
	}
	if err := newConfig.Validate(); err != nil {
		cm.mu.Unlock()
		return err
	}
	cm.oldConfig = cm.config
	cm.config = newConfig
	configCopy := cm.config
	oldConfigCopy := cm.oldConfig
	subscribers := append([]func(NewConfig Config, oldConfig Config){}, cm.subscribers...)
	cm.mu.Unlock()

	// 通知订阅者
	for _, sub := range subscribers {
		sub(configCopy, oldConfigCopy)
	}

	return cm.Save()
}

func (cm *ConfigManager) applyRealtimeDefaults() {
	defaults := NewDefaultConfig()
	if cm.config.RealtimeRegion == "" {
		cm.config.RealtimeRegion = defaults.RealtimeRegion
	}
	if cm.config.RealtimeModel == "" {
		cm.config.RealtimeModel = defaults.RealtimeModel
	}
	if cm.config.RealtimePrompt == "" {
		cm.config.RealtimePrompt = defaults.RealtimePrompt
	}
	if cm.config.RealtimeTopP == 0 {
		cm.config.RealtimeTopP = defaults.RealtimeTopP
	}
	if cm.config.RealtimeMaxTokens == 0 {
		cm.config.RealtimeMaxTokens = defaults.RealtimeMaxTokens
	}
	if cm.config.RealtimeVADType == "" {
		cm.config.RealtimeVADType = defaults.RealtimeVADType
	}
	if cm.config.RealtimeSilenceDurationMs == 0 {
		cm.config.RealtimeSilenceDurationMs = defaults.RealtimeSilenceDurationMs
	}
}

// migrateLegacyRealtimeDefaults 只迁移仍使用旧内置提示词和旧默认参数的配置。
// 用户修改过的提示词或参数保持原值。
func (cm *ConfigManager) migrateLegacyRealtimeDefaults() {
	if cm.config.RealtimePrompt != legacyDefaultRealtimePrompt {
		return
	}
	cm.config.RealtimePrompt = DefaultRealtimePrompt
	if cm.config.RealtimeTemperature == 0.3 {
		cm.config.RealtimeTemperature = 0.4
	}
	if cm.config.RealtimeTopK == 0 {
		cm.config.RealtimeTopK = 20
	}
	if cm.config.RealtimeMaxTokens == 2000 {
		cm.config.RealtimeMaxTokens = 600
	}
}

func (cm *ConfigManager) Subscribe(callback func(NewConfig Config, oldConfig Config)) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.subscribers = append(cm.subscribers, callback)
}
