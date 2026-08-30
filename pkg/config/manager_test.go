package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPartialUpdatesKeepScreenshotAndRealtimeSettingsIsolated(t *testing.T) {
	cm := NewConfigManager()
	cm.configPath = filepath.Join(t.TempDir(), "config.json")
	cm.config.RealtimeAPIKey = "voice-secret"
	cm.config.RealtimeWorkspaceID = "workspace-1"
	cm.config.APIKey = "screenshot-secret"
	cm.config.Model = "screenshot-model"
	cm.config.RAGEmbeddingModel = "qwen3.7-text-embedding"
	cm.config.RAGWorkspaceID = "rag-workspace"

	if err := cm.UpdateFromJSON(`{"model":"new-screenshot-model"}`); err != nil {
		t.Fatal(err)
	}
	got := cm.Get()
	if got.RealtimeAPIKey != "voice-secret" || got.RealtimeWorkspaceID != "workspace-1" {
		t.Fatalf("screenshot update overwrote realtime settings: %+v", got)
	}
	if got.RAGEmbeddingModel != "qwen3.7-text-embedding" || got.RAGWorkspaceID != "rag-workspace" {
		t.Fatalf("screenshot update overwrote RAG settings: %+v", got)
	}

	if err := cm.UpdateFromJSON(`{"realtimeModel":"new-voice-model","realtimePrompt":"voice prompt"}`); err != nil {
		t.Fatal(err)
	}
	got = cm.Get()
	if got.APIKey != "screenshot-secret" || got.Model != "new-screenshot-model" {
		t.Fatalf("realtime update overwrote screenshot settings: %+v", got)
	}

	if err := cm.UpdateFromJSON(`{"ragRetrievalMode":"local","ragTopK":8}`); err != nil {
		t.Fatal(err)
	}
	got = cm.Get()
	if got.APIKey != "screenshot-secret" || got.RealtimeAPIKey != "voice-secret" {
		t.Fatalf("RAG update overwrote screenshot or realtime settings: %+v", got)
	}
}

func TestOldConfigLoadsWithRealtimeDefaults(t *testing.T) {
	cm := NewConfigManager()
	dir := t.TempDir()
	cm.configPath = filepath.Join(dir, "config.json")
	if err := os.WriteFile(cm.configPath, []byte(`{"apiKey":"old-key","model":"old-model","prompt":"old-prompt"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := cm.Load(); err != nil {
		t.Fatal(err)
	}
	got := cm.Get()
	if got.APIKey != "old-key" || got.Model != "old-model" || got.Prompt != "old-prompt" {
		t.Fatalf("old screenshot config changed: %+v", got)
	}
	if got.RealtimeModel != "qwen3.5-omni-plus-realtime" || got.RealtimePrompt == "" {
		t.Fatalf("realtime defaults missing: %+v", got)
	}
	if got.RAGEnabled || got.RAGRetrievalMode != "hybrid" || got.RAGEmbeddingModel != "qwen3.7-text-embedding" || got.RAGEmbeddingDimensions != 1024 {
		t.Fatalf("RAG defaults missing or unexpectedly enabled: %+v", got)
	}
	if got.Theme != "dark" {
		t.Fatalf("旧配置没有使用夜间主题默认值: theme=%q", got.Theme)
	}
}

func TestRAGValidationRejectsInvalidModeAndDimensions(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.RAGRetrievalMode = "unknown"
	if err := cfg.Validate(); err == nil {
		t.Fatal("invalid RAG mode should be rejected")
	}
	cfg = NewDefaultConfig()
	cfg.RAGEmbeddingDimensions = 0
	if err := cfg.Validate(); err == nil {
		t.Fatal("invalid embedding dimensions should be rejected")
	}
}

func TestThemePersistsWithoutOverwritingOtherSettings(t *testing.T) {
	cm := NewConfigManager()
	cm.configPath = filepath.Join(t.TempDir(), "config.json")
	cm.config.APIKey = "screenshot-secret"
	cm.config.RealtimeWorkspaceID = "workspace-1"

	if err := cm.UpdateFromJSON(`{"theme":"light"}`); err != nil {
		t.Fatal(err)
	}
	got := cm.Get()
	if got.Theme != "light" {
		t.Fatalf("白天主题没有保存: theme=%q", got.Theme)
	}
	if got.APIKey != "screenshot-secret" || got.RealtimeWorkspaceID != "workspace-1" {
		t.Fatalf("保存主题覆盖了其他配置: %+v", got)
	}

	reloaded := NewConfigManager()
	reloaded.configPath = cm.configPath
	if err := reloaded.Load(); err != nil {
		t.Fatal(err)
	}
	if reloaded.Get().Theme != "light" {
		t.Fatalf("重新加载后主题丢失: theme=%q", reloaded.Get().Theme)
	}
}

func TestInvalidThemeIsRejected(t *testing.T) {
	cm := NewConfigManager()
	cm.configPath = filepath.Join(t.TempDir(), "config.json")
	if err := cm.UpdateFromJSON(`{"theme":"blue"}`); err == nil {
		t.Fatal("expected invalid theme validation error")
	}
}

func TestRealtimeAndSTTCannotBothBeEnabled(t *testing.T) {
	cm := NewConfigManager()
	cm.configPath = filepath.Join(t.TempDir(), "config.json")
	if err := cm.UpdateFromJSON(`{"realtimeEnabled":true,"sttEnabled":true}`); err == nil {
		t.Fatal("expected mutually exclusive realtime/STT validation error")
	}
}

func TestRealtimeZeroValuesSurviveUnrelatedPartialUpdate(t *testing.T) {
	cm := NewConfigManager()
	cm.configPath = filepath.Join(t.TempDir(), "config.json")
	cm.config.RealtimeTemperature = 0
	cm.config.RealtimeVADThreshold = 0

	if err := cm.UpdateFromJSON(`{"model":"unchanged-voice-config"}`); err != nil {
		t.Fatal(err)
	}
	got := cm.Get()
	if got.RealtimeTemperature != 0 || got.RealtimeVADThreshold != 0 {
		t.Fatalf("合法的语音零值被覆盖: temperature=%v threshold=%v", got.RealtimeTemperature, got.RealtimeVADThreshold)
	}
}

func TestLegacyRealtimeDefaultsMigrateToConciseOralDefaults(t *testing.T) {
	cm := NewConfigManager()
	cm.configPath = filepath.Join(t.TempDir(), "config.json")
	legacy := NewDefaultConfig()
	legacy.RealtimePrompt = legacyDefaultRealtimePrompt
	legacy.RealtimeTemperature = 0.3
	legacy.RealtimeTopK = 0
	legacy.RealtimeMaxTokens = 2000
	data, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cm.configPath, data, 0600); err != nil {
		t.Fatal(err)
	}
	if err := cm.Load(); err != nil {
		t.Fatal(err)
	}
	got := cm.Get()
	if got.RealtimePrompt != DefaultRealtimePrompt {
		t.Fatal("旧内置语音提示词没有迁移到口语化短回答版本")
	}
	if got.RealtimeTemperature != 0.4 || got.RealtimeTopP != 0.8 || got.RealtimeTopK != 20 || got.RealtimeMaxTokens != 600 {
		t.Fatalf("旧语音参数没有迁移到推荐值: temperature=%v topP=%v topK=%v maxTokens=%v", got.RealtimeTemperature, got.RealtimeTopP, got.RealtimeTopK, got.RealtimeMaxTokens)
	}
	for _, phrase := range []string{"直接说出口", "30 到 60 秒", "不要使用 Markdown 标题"} {
		if !strings.Contains(got.RealtimePrompt, phrase) {
			t.Fatalf("新提示词缺少长度或口语约束 %q", phrase)
		}
	}
}

func TestCustomRealtimePromptAndParametersAreNotMigrated(t *testing.T) {
	cm := NewConfigManager()
	cm.configPath = filepath.Join(t.TempDir(), "config.json")
	custom := NewDefaultConfig()
	custom.RealtimePrompt = "我的自定义面试提示词"
	custom.RealtimeTemperature = 0.7
	custom.RealtimeTopK = 7
	custom.RealtimeMaxTokens = 900
	data, err := json.Marshal(custom)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cm.configPath, data, 0600); err != nil {
		t.Fatal(err)
	}
	if err := cm.Load(); err != nil {
		t.Fatal(err)
	}
	got := cm.Get()
	if got.RealtimePrompt != custom.RealtimePrompt || got.RealtimeTemperature != 0.7 || got.RealtimeTopK != 7 || got.RealtimeMaxTokens != 900 {
		t.Fatalf("用户自定义语音配置被覆盖: %+v", got)
	}
}

func TestConfigPathUsesStableUserConfigDirectory(t *testing.T) {
	got := configPathFor(filepath.Join("C:", "Users", "tester", "AppData", "Roaming"))
	want := filepath.Join("C:", "Users", "tester", "AppData", "Roaming", "Q-Solver", "config.json")
	if got != want {
		t.Fatalf("配置路径不稳定: got=%q want=%q", got, want)
	}
}

func TestMigrateLegacyConfigCopiesValidJSONWithoutOverwriting(t *testing.T) {
	dir := t.TempDir()
	legacyPath := filepath.Join(dir, "legacy", "config.json")
	targetPath := filepath.Join(dir, "stable", "config.json")
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0700); err != nil {
		t.Fatal(err)
	}
	legacyData := []byte(`{"model":"legacy-model","shortcuts":{"toggle":{"vkCode":"120","keyName":"F9"}}}`)
	if err := os.WriteFile(legacyPath, legacyData, 0600); err != nil {
		t.Fatal(err)
	}

	if err := migrateLegacyConfig(targetPath, []string{filepath.Join(dir, "missing.json"), legacyPath}); err != nil {
		t.Fatal(err)
	}
	migrated, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(migrated) != string(legacyData) {
		t.Fatalf("旧配置迁移内容改变: got=%s", migrated)
	}

	if err := os.WriteFile(targetPath, []byte(`{"model":"user-newer-model"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := migrateLegacyConfig(targetPath, []string{legacyPath}); err != nil {
		t.Fatal(err)
	}
	kept, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(kept) != `{"model":"user-newer-model"}` {
		t.Fatalf("已有稳定配置被覆盖: %s", kept)
	}
}

func TestMigrateLegacyConfigSkipsInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	invalidPath := filepath.Join(dir, "invalid.json")
	targetPath := filepath.Join(dir, "stable", "config.json")
	if err := os.WriteFile(invalidPath, []byte("not-json"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := migrateLegacyConfig(targetPath, []string{invalidPath}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(targetPath); !os.IsNotExist(err) {
		t.Fatalf("无效旧配置不应迁移: %v", err)
	}
}

func TestFullyTransparentOpacityPersists(t *testing.T) {
	cm := NewConfigManager()
	cm.configPath = filepath.Join(t.TempDir(), "config.json")
	cm.legacyPaths = nil
	cm.config.Opacity = 0
	if err := cm.Save(); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	if _, ok := raw["opacity"]; !ok {
		t.Fatal("opacity=0 被 omitempty 丢弃，透明度无法持久化")
	}

	reloaded := NewConfigManager()
	reloaded.configPath = cm.configPath
	reloaded.legacyPaths = nil
	if err := reloaded.Load(); err != nil {
		t.Fatal(err)
	}
	if reloaded.Get().Opacity != 0 {
		t.Fatalf("完全透明设置重新加载后改变: %v", reloaded.Get().Opacity)
	}
}

func TestLegacyAITextOpacityMigratesAsUserVisibleTransparency(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(configPath, []byte(`{"aiTextOpacity":0.95}`), 0600); err != nil {
		t.Fatal(err)
	}

	cm := NewConfigManager()
	cm.configPath = configPath
	cm.legacyPaths = nil
	if err := cm.Load(); err != nil {
		t.Fatal(err)
	}
	if got := cm.Get().AITextTransparency; got != 0.95 {
		t.Fatalf("旧版滑块值没有按透明度语义迁移: got=%v want=0.95", got)
	}
}

func TestAITextColorDefaultsToWhiteForLegacyConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(configPath, []byte(`{"theme":"dark"}`), 0600); err != nil {
		t.Fatal(err)
	}

	cm := NewConfigManager()
	cm.configPath = configPath
	cm.legacyPaths = nil
	if err := cm.Load(); err != nil {
		t.Fatal(err)
	}
	if got := cm.Get().AITextColor; got != "white" {
		t.Fatalf("旧配置应默认使用白色 AI 文字: got=%q", got)
	}
}

func TestAITextColorPersistsBlack(t *testing.T) {
	cm := NewConfigManager()
	cm.configPath = filepath.Join(t.TempDir(), "config.json")
	cm.legacyPaths = nil
	if err := cm.UpdateFromJSON(`{"aiTextColor":"black"}`); err != nil {
		t.Fatal(err)
	}

	reloaded := NewConfigManager()
	reloaded.configPath = cm.configPath
	reloaded.legacyPaths = nil
	if err := reloaded.Load(); err != nil {
		t.Fatal(err)
	}
	if got := reloaded.Get().AITextColor; got != "black" {
		t.Fatalf("黑色 AI 文字没有正确持久化: got=%q", got)
	}
}

func TestAITextColorRejectsUnsupportedValues(t *testing.T) {
	config := NewDefaultConfig()
	config.AITextColor = "green"
	if err := config.Validate(); err == nil {
		t.Fatal("不支持的 AI 文字颜色应被拒绝")
	}
}
