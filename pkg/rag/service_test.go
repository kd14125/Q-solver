package rag

import (
	"Q-Solver/pkg/config"
	"archive/zip"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func configForTest() config.Config {
	cfg := config.NewDefaultConfig()
	cfg.APIKey = "screenshot-key"
	cfg.RealtimeAPIKey = "voice-key"
	cfg.RealtimeWorkspaceID = "voice-workspace"
	return cfg
}

func testSettings() Settings {
	return Settings{Mode: ModeLocal, EmbeddingModel: "qwen3.7-text-embedding", Dimensions: 4, TopK: 5, MaxContextChars: 6000}
}

func newTestService(t *testing.T) *Service {
	t.Helper()
	service, err := NewService(filepath.Join(t.TempDir(), "knowledge.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { service.Close() })
	return service
}

func TestLocalSearchUsesChineseNGramsWithoutEmbeddingRequest(t *testing.T) {
	service := newTestService(t)
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		http.Error(w, "must not be called", http.StatusInternalServerError)
	}))
	defer server.Close()
	service.SetHTTPClient(server.Client())
	settings := testSettings()
	settings.BaseURL = server.URL
	settings.APIKey = "unused"

	if _, err := service.AddQA(context.Background(), "如何保证 WebSocket 并发写安全？", "所有写操作统一经过互斥锁。", settings); err != nil {
		t.Fatal(err)
	}
	result, err := service.Search(context.Background(), "WebSocket并发写入怎么保证安全", settings)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Hits) == 0 || !strings.Contains(result.Hits[0].Content, "互斥锁") {
		t.Fatalf("中文本地检索未命中: %+v", result)
	}
	if requests.Load() != 0 {
		t.Fatalf("本地模式不应调用 Embedding API: %d", requests.Load())
	}
}

func TestAPIAndHybridSearchUseQwenEmbeddingAndFallback(t *testing.T) {
	service := newTestService(t)
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.URL.Path != "/embeddings" || r.Header.Get("Authorization") != "Bearer voice-key" {
			t.Fatalf("unexpected embedding request: %s auth=%q", r.URL.Path, r.Header.Get("Authorization"))
		}
		var body struct {
			Model      string   `json:"model"`
			Input      []string `json:"input"`
			Dimensions int      `json:"dimensions"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Model != "qwen3.7-text-embedding" || body.Dimensions != 4 {
			t.Fatalf("unexpected embedding settings: %+v", body)
		}
		data := make([]map[string]any, len(body.Input))
		for index := range body.Input {
			data[index] = map[string]any{"index": index, "embedding": []float32{1, 0, 0, 0}}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
	}))
	defer server.Close()
	service.SetHTTPClient(server.Client())
	settings := testSettings()
	settings.Mode = ModeHybrid
	settings.BaseURL = server.URL
	settings.APIKey = "voice-key"

	if _, err := service.AddQA(context.Background(), "介绍你负责的实时语音项目", "我负责音频采集、VAD 与实时回答链路。", settings); err != nil {
		t.Fatal(err)
	}
	result, err := service.Search(context.Background(), "说说实时面试助手的经历", settings)
	if err != nil || len(result.Hits) == 0 {
		t.Fatalf("hybrid search failed: result=%+v err=%v", result, err)
	}
	if requests.Load() < 2 {
		t.Fatalf("expected indexing and query embedding requests, got %d", requests.Load())
	}

	service.SetHTTPClient(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, context.DeadlineExceeded
	})})
	fallback, err := service.Search(context.Background(), "实时语音项目", settings)
	if err != nil || len(fallback.Hits) == 0 || !strings.Contains(fallback.Warning, "API") {
		t.Fatalf("hybrid should fall back to local results: %+v err=%v", fallback, err)
	}
}

func TestImportDeduplicatesAndModelChangeInvalidatesVectors(t *testing.T) {
	service := newTestService(t)
	path := filepath.Join(t.TempDir(), "project.md")
	if err := os.WriteFile(path, []byte("我负责 Q-Solver 的 Windows 音频回环采集和稳定性优化。"), 0600); err != nil {
		t.Fatal(err)
	}
	settings := testSettings()
	first, err := service.ImportFile(context.Background(), path, settings)
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.ImportFile(context.Background(), path, settings)
	if err != nil {
		t.Fatal(err)
	}
	if first.Document.ID == 0 || !second.Duplicated || second.Document.ID != first.Document.ID {
		t.Fatalf("duplicate import not detected: first=%+v second=%+v", first, second)
	}
}

func TestModelChangeMarksVectorsPendingAndDocumentDeleteRemovesChunks(t *testing.T) {
	service := newTestService(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Input []string `json:"input"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		data := make([]map[string]any, len(body.Input))
		for index := range body.Input {
			data[index] = map[string]any{"index": index, "embedding": []float32{1, 0, 0, 0}}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
	}))
	defer server.Close()
	service.SetHTTPClient(server.Client())
	settings := testSettings()
	settings.Mode = ModeHybrid
	settings.BaseURL = server.URL
	settings.APIKey = "test-key"
	path := filepath.Join(t.TempDir(), "experience.txt")
	if err := os.WriteFile(path, []byte("我负责桌面端系统音频回环采集。"), 0600); err != nil {
		t.Fatal(err)
	}
	imported, err := service.ImportFile(context.Background(), path, settings)
	if err != nil || imported.Warning != "" {
		t.Fatalf("import failed: %+v err=%v", imported, err)
	}
	snapshot, err := service.List(context.Background(), settings)
	if err != nil || len(snapshot.Documents) != 1 || snapshot.Documents[0].Status != "ready" {
		t.Fatalf("vector should be ready: %+v err=%v", snapshot, err)
	}

	changed := settings
	changed.EmbeddingModel = "another-embedding-model"
	snapshot, err = service.List(context.Background(), changed)
	if err != nil || snapshot.Documents[0].Status != "pending" {
		t.Fatalf("model change should invalidate old vectors: %+v err=%v", snapshot, err)
	}
	apiResult, err := service.Search(context.Background(), "音频采集", Settings{Mode: ModeAPI, APIKey: changed.APIKey, BaseURL: changed.BaseURL, EmbeddingModel: changed.EmbeddingModel, Dimensions: 4, TopK: 5})
	if err != nil || len(apiResult.Hits) != 0 {
		t.Fatalf("API mode must not use vectors from another model: %+v err=%v", apiResult, err)
	}

	if err := service.DeleteDocument(context.Background(), imported.Document.ID); err != nil {
		t.Fatal(err)
	}
	localResult, err := service.Search(context.Background(), "音频采集", testSettings())
	if err != nil || len(localResult.Hits) != 0 {
		t.Fatalf("document chunks were not deleted: %+v err=%v", localResult, err)
	}
}

func TestDOCXTextExtraction(t *testing.T) {
	path := filepath.Join(t.TempDir(), "resume.docx")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	archive := zip.NewWriter(file)
	part, err := archive.Create("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write([]byte(`<?xml version="1.0"?><w:document xmlns:w="x"><w:body><w:p><w:r><w:t>负责 Q-Solver 实时语音功能</w:t></w:r></w:p></w:body></w:document>`))
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	kind, content, err := extractFile(path)
	if err != nil || kind != "docx" || !strings.Contains(content, "实时语音") {
		t.Fatalf("DOCX extraction failed: kind=%q content=%q err=%v", kind, content, err)
	}
}

func TestSettingsInheritOnlyRealtimeCredentials(t *testing.T) {
	cfg := configForTest()
	settings := SettingsFromConfig(cfg)
	if settings.APIKey != "voice-key" || settings.WorkspaceID != "voice-workspace" {
		t.Fatalf("RAG did not inherit realtime credentials: %+v", settings)
	}
	if strings.Contains(settings.APIKey, "screenshot") {
		t.Fatal("RAG must not inherit screenshot credentials")
	}
	baseURL, err := settings.EmbeddingBaseURL()
	if err != nil || baseURL != "https://voice-workspace.cn-beijing.maas.aliyuncs.com/compatible-mode/v1" {
		t.Fatalf("unexpected embedding base URL: %q err=%v", baseURL, err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return fn(req) }
