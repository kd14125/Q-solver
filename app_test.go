package main

import (
	"Q-Solver/pkg/config"
	"testing"
)

func TestScreenshotSettingsDoNotReconnectActiveQwenSession(t *testing.T) {
	oldConfig := config.NewDefaultConfig()
	oldConfig.UseLiveApi = true
	oldConfig.RealtimeEnabled = true
	newConfig := oldConfig
	newConfig.APIKey = "new-screenshot-key"
	newConfig.Model = "new-screenshot-model"
	newConfig.Prompt = "new-screenshot-prompt"

	if voiceSessionConfigChanged(newConfig, oldConfig) {
		t.Fatal("修改截图答题配置不应重连 Qwen Realtime 会话")
	}
}

func TestRealtimeSettingsReconnectActiveQwenSession(t *testing.T) {
	oldConfig := config.NewDefaultConfig()
	oldConfig.UseLiveApi = true
	oldConfig.RealtimeEnabled = true
	newConfig := oldConfig
	newConfig.RealtimePrompt = "updated voice prompt"

	if !voiceSessionConfigChanged(newConfig, oldConfig) {
		t.Fatal("修改语音配置应重连 Qwen Realtime 会话")
	}
}

func TestScreenshotSettingsStillReconnectGeminiLiveSession(t *testing.T) {
	oldConfig := config.NewDefaultConfig()
	oldConfig.UseLiveApi = true
	newConfig := oldConfig
	newConfig.Model = "updated-gemini-live-model"

	if !voiceSessionConfigChanged(newConfig, oldConfig) {
		t.Fatal("原 Gemini Live 模式应继续响应截图模型配置变化")
	}
}
