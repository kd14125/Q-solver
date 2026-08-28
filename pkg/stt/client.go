package stt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

// Client 调用 OpenAI-compatible 的语音转文字接口。
type Client struct {
	APIKey   string
	BaseURL  string
	Model    string
	Language string
	HTTP     *http.Client
}

// GetModels 获取 OpenAI-compatible 语音识别模型列表。
// 大多数中转站会在 BaseURL 下提供 GET /models，并返回 {"data":[{"id":"..."}]}。
func (c *Client) GetModels(ctx context.Context) ([]string, error) {
	if strings.TrimSpace(c.APIKey) == "" {
		return nil, fmt.Errorf("语音转文字 API Key 不能为空")
	}
	if strings.TrimSpace(c.BaseURL) == "" {
		return nil, fmt.Errorf("语音转文字 API 地址不能为空")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, modelsURL(c.BaseURL), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Accept", "application/json")

	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("获取语音识别模型失败: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("模型列表接口返回 %s: %s", resp.Status, strings.TrimSpace(string(data)))
	}

	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("无法解析语音识别模型列表: %w", err)
	}

	models := make([]string, 0, len(result.Data))
	seen := make(map[string]struct{}, len(result.Data))
	for _, item := range result.Data {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		models = append(models, id)
	}
	return models, nil
}

func (c *Client) Transcribe(ctx context.Context, wav []byte) (string, error) {
	if strings.TrimSpace(c.APIKey) == "" {
		return "", fmt.Errorf("语音转文字 API Key 不能为空")
	}
	if strings.TrimSpace(c.BaseURL) == "" {
		return "", fmt.Errorf("语音转文字 API 地址不能为空")
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	file, err := writer.CreateFormFile("file", "segment.wav")
	if err != nil {
		return "", err
	}
	if _, err = file.Write(wav); err != nil {
		return "", err
	}
	model := strings.TrimSpace(c.Model)
	if model == "" {
		model = "whisper-1"
	}
	_ = writer.WriteField("model", model)
	_ = writer.WriteField("response_format", "json")
	if language := strings.TrimSpace(c.Language); language != "" {
		_ = writer.WriteField("language", language)
	}
	if err = writer.Close(); err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, transcriptionURL(c.BaseURL), &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 90 * time.Second}
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("语音转文字请求失败: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("语音转文字接口返回 %s: %s", resp.Status, strings.TrimSpace(string(data)))
	}
	var result struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("无法解析语音转文字响应: %w", err)
	}
	return strings.TrimSpace(result.Text), nil
}

func transcriptionURL(baseURL string) string {
	url := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if strings.HasSuffix(strings.ToLower(url), "/audio/transcriptions") {
		return url
	}
	return url + "/audio/transcriptions"
}

func modelsURL(baseURL string) string {
	url := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if strings.HasSuffix(strings.ToLower(url), "/models") {
		return url
	}
	if strings.HasSuffix(strings.ToLower(url), "/audio/transcriptions") {
		return url[:len(url)-len("/audio/transcriptions")] + "/models"
	}
	return url + "/models"
}

// PCM16MonoToWAV 为 16kHz、16-bit、单声道 PCM 添加 WAV 文件头。
func PCM16MonoToWAV(pcm []byte, sampleRate int) []byte {
	dataSize := uint32(len(pcm))
	byteRate := uint32(sampleRate * 2)
	out := make([]byte, 44+len(pcm))
	copy(out[0:4], "RIFF")
	putLE32(out[4:8], 36+dataSize)
	copy(out[8:12], "WAVE")
	copy(out[12:16], "fmt ")
	putLE32(out[16:20], 16)
	putLE16(out[20:22], 1)
	putLE16(out[22:24], 1)
	putLE32(out[24:28], uint32(sampleRate))
	putLE32(out[28:32], byteRate)
	putLE16(out[32:34], 2)
	putLE16(out[34:36], 16)
	copy(out[36:40], "data")
	putLE32(out[40:44], dataSize)
	copy(out[44:], pcm)
	return out
}

func putLE16(dst []byte, value uint16) {
	dst[0], dst[1] = byte(value), byte(value>>8)
}

func putLE32(dst []byte, value uint32) {
	dst[0], dst[1], dst[2], dst[3] = byte(value), byte(value>>8), byte(value>>16), byte(value>>24)
}
