package rag

import (
	"Q-Solver/pkg/config"
	"fmt"
	"strings"
)

const (
	ModeLocal  = "local"
	ModeAPI    = "api"
	ModeHybrid = "hybrid"
)

type Settings struct {
	Enabled         bool   `json:"enabled"`
	Mode            string `json:"mode"`
	APIKey          string `json:"-"`
	WorkspaceID     string `json:"workspaceID,omitempty"`
	Region          string `json:"region,omitempty"`
	BaseURL         string `json:"baseURL,omitempty"`
	EmbeddingModel  string `json:"embeddingModel"`
	Dimensions      int    `json:"dimensions"`
	TopK            int    `json:"topK"`
	MaxContextChars int    `json:"maxContextChars"`
}

// SettingsFromConfig 仅继承语音 Realtime 凭据，禁止回退到截图答题配置。
func SettingsFromConfig(cfg config.Config) Settings {
	apiKey := strings.TrimSpace(cfg.RAGAPIKey)
	if apiKey == "" {
		apiKey = strings.TrimSpace(cfg.RealtimeAPIKey)
	}
	workspaceID := strings.TrimSpace(cfg.RAGWorkspaceID)
	if workspaceID == "" {
		workspaceID = strings.TrimSpace(cfg.RealtimeWorkspaceID)
	}
	region := strings.TrimSpace(cfg.RAGRegion)
	if region == "" {
		region = strings.TrimSpace(cfg.RealtimeRegion)
	}
	if region == "" {
		region = "cn-beijing"
	}
	return Settings{
		Enabled:         cfg.RAGEnabled,
		Mode:            cfg.RAGRetrievalMode,
		APIKey:          apiKey,
		WorkspaceID:     workspaceID,
		Region:          region,
		BaseURL:         strings.TrimSpace(cfg.RAGBaseURL),
		EmbeddingModel:  cfg.RAGEmbeddingModel,
		Dimensions:      cfg.RAGEmbeddingDimensions,
		TopK:            cfg.RAGTopK,
		MaxContextChars: cfg.RAGMaxContextChars,
	}
}

func (s Settings) EmbeddingBaseURL() (string, error) {
	if strings.TrimSpace(s.BaseURL) != "" {
		return strings.TrimRight(strings.TrimSpace(s.BaseURL), "/"), nil
	}
	if strings.TrimSpace(s.WorkspaceID) == "" {
		return "", fmt.Errorf("RAG Workspace ID 不能为空")
	}
	region := s.Region
	if region == "" {
		region = "cn-beijing"
	}
	return fmt.Sprintf("https://%s.%s.maas.aliyuncs.com/compatible-mode/v1", s.WorkspaceID, region), nil
}

type Document struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Path       string `json:"path,omitempty"`
	Kind       string `json:"kind"`
	Status     string `json:"status"`
	ChunkCount int    `json:"chunkCount"`
	CreatedAt  int64  `json:"createdAt"`
	UpdatedAt  int64  `json:"updatedAt"`
}

type QAEntry struct {
	ID        int64  `json:"id"`
	Question  string `json:"question"`
	Answer    string `json:"answer"`
	Status    string `json:"status"`
	Warning   string `json:"warning,omitempty"`
	CreatedAt int64  `json:"createdAt"`
	UpdatedAt int64  `json:"updatedAt"`
}

type Snapshot struct {
	Documents []Document `json:"documents"`
	QAEntries []QAEntry  `json:"qaEntries"`
}

type ImportResult struct {
	Path       string   `json:"path"`
	Document   Document `json:"document"`
	Warning    string   `json:"warning,omitempty"`
	Duplicated bool     `json:"duplicated"`
}

type SearchHit struct {
	ID         int64   `json:"id"`
	Kind       string  `json:"kind"`
	Title      string  `json:"title"`
	Content    string  `json:"content"`
	Source     string  `json:"source"`
	Score      float64 `json:"score"`
	LocalRank  int     `json:"localRank,omitempty"`
	VectorRank int     `json:"vectorRank,omitempty"`
}

type SearchResult struct {
	Mode       string      `json:"mode"`
	Hits       []SearchHit `json:"hits"`
	Warning    string      `json:"warning,omitempty"`
	DurationMs int64       `json:"durationMs"`
}

type SearchTestResult struct {
	Local  SearchResult `json:"local"`
	API    SearchResult `json:"api"`
	Hybrid SearchResult `json:"hybrid"`
}

type IndexResult struct {
	Total   int    `json:"total"`
	Indexed int    `json:"indexed"`
	Warning string `json:"warning,omitempty"`
}
