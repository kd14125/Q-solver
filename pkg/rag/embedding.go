package rag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Embedder interface {
	Embed(context.Context, []string) ([][]float32, error)
}

type EmbeddingClient struct {
	settings Settings
	client   *http.Client
}

func NewEmbeddingClient(settings Settings, client *http.Client) *EmbeddingClient {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &EmbeddingClient{settings: settings, client: client}
}

func (c *EmbeddingClient) Embed(ctx context.Context, inputs []string) ([][]float32, error) {
	if len(inputs) == 0 {
		return nil, nil
	}
	if strings.TrimSpace(c.settings.APIKey) == "" {
		return nil, fmt.Errorf("RAG API Key 不能为空")
	}
	baseURL, err := c.settings.EmbeddingBaseURL()
	if err != nil {
		return nil, err
	}
	payload := map[string]any{
		"model":           c.settings.EmbeddingModel,
		"input":           inputs,
		"dimensions":      c.settings.Dimensions,
		"encoding_format": "float",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(baseURL, "/")+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.settings.APIKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Embedding API 请求失败: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Embedding API 返回 HTTP %d", resp.StatusCode)
	}
	var decoded struct {
		Data []struct {
			Index     int       `json:"index"`
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return nil, fmt.Errorf("解析 Embedding API 响应失败: %w", err)
	}
	if decoded.Error.Message != "" {
		return nil, fmt.Errorf("Embedding API 返回错误: %s", decoded.Error.Message)
	}
	if len(decoded.Data) != len(inputs) {
		return nil, fmt.Errorf("Embedding API 返回向量数量不匹配")
	}
	result := make([][]float32, len(inputs))
	for _, item := range decoded.Data {
		if item.Index < 0 || item.Index >= len(result) || len(item.Embedding) != c.settings.Dimensions {
			return nil, fmt.Errorf("Embedding API 返回了无效向量")
		}
		result[item.Index] = item.Embedding
	}
	return result, nil
}
