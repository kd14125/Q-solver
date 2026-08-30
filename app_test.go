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

func TestRAGSettingsReconnectOnlyActiveQwenSession(t *testing.T) {
	oldConfig := config.NewDefaultConfig()
	oldConfig.UseLiveApi = true
	oldConfig.RealtimeEnabled = true
	newConfig := oldConfig
	newConfig.RAGEnabled = true
	newConfig.RAGRetrievalMode = "hybrid"

	if !voiceSessionConfigChanged(newConfig, oldConfig) {
		t.Fatal("修改 RAG 配置应重连正在运行的 Qwen Realtime 会话")
	}

	oldGemini := config.NewDefaultConfig()
	oldGemini.UseLiveApi = true
	newGemini := oldGemini
	newGemini.RAGEnabled = true
	if voiceSessionConfigChanged(newGemini, oldGemini) {
		t.Fatal("RAG 配置不应影响 Gemini Live 或截图模型链路")
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

func TestScreenshotAPIStaysAvailableDuringRealtimeInterview(t *testing.T) {
	cfg := config.NewDefaultConfig()
	cfg.APIKey = "screenshot-key"
	cfg.UseLiveApi = true
	cfg.RealtimeEnabled = true

	if !screenshotAPIAvailable(cfg) {
		t.Fatal("开启 Qwen Realtime 时不应禁用独立的截图答题 API")
	}
}

func TestScreenshotAPIStillRequiresItsOwnKey(t *testing.T) {
	cfg := config.NewDefaultConfig()
	cfg.UseLiveApi = true
	cfg.RealtimeEnabled = true

	if screenshotAPIAvailable(cfg) {
		t.Fatal("截图答题 API 未配置自己的 API Key 时不应可用")
	}
}
