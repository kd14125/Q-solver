package llm

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const qwenRealtimeTranscriptionModel = "qwen3-asr-flash-realtime"

// QwenRealtimeProvider 实现 Qwen3.5 Omni Realtime 的独立实时连接。
type QwenRealtimeProvider struct {
	dialer *websocket.Dialer
}

func NewQwenRealtimeProvider() *QwenRealtimeProvider {
	return &QwenRealtimeProvider{dialer: websocket.DefaultDialer}
}

func (p *QwenRealtimeProvider) ConnectLive(ctx context.Context, cfg *LiveConfig) (LiveSession, error) {
	if cfg == nil {
		return nil, errors.New("语音实时配置不能为空")
	}
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, errors.New("Realtime API Key 不能为空")
	}
	endpoint, err := buildQwenRealtimeURL(cfg)
	if err != nil {
		return nil, err
	}

	header := http.Header{}
	header.Set("Authorization", "Bearer "+cfg.APIKey)
	conn, response, err := p.dialer.DialContext(ctx, endpoint, header)
	if err != nil {
		if response != nil {
			return nil, fmt.Errorf("Qwen Realtime 连接失败（HTTP %d）", response.StatusCode)
		}
		return nil, fmt.Errorf("Qwen Realtime 连接失败: %w", err)
	}

	session := &QwenRealtimeSession{
		conn:             conn,
		closed:           make(chan struct{}),
		completedItems:   make(map[string]struct{}),
		completedReplies: make(map[string]struct{}),
		toolResponseIDs:  make(map[string]struct{}),
	}
	if err := session.sendJSON(buildQwenSessionUpdate(cfg)); err != nil {
		session.Close()
		return nil, fmt.Errorf("发送 session.update 失败: %w", err)
	}

	// 连接建立以 session.updated 为准，避免“测试连接”只验证到 WebSocket 握手。
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetReadDeadline(deadline)
	} else {
		_ = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	}
	for {
		_, data, readErr := conn.ReadMessage()
		if readErr != nil {
			session.Close()
			return nil, fmt.Errorf("等待 session.updated 失败: %w", readErr)
		}
		var event qwenServerEvent
		if err := json.Unmarshal(data, &event); err != nil {
			continue
		}
		switch event.Type {
		case "session.updated":
			_ = conn.SetReadDeadline(time.Time{})
			return session, nil
		case "error":
			session.Close()
			return nil, errors.New(qwenEventError(event))
		}
	}
}

func buildQwenRealtimeURL(cfg *LiveConfig) (string, error) {
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = "qwen3.5-omni-plus-realtime"
	}
	endpoint := strings.TrimSpace(cfg.BaseURL)
	if endpoint == "" {
		workspaceID := strings.TrimSpace(cfg.WorkspaceID)
		if workspaceID == "" {
			return "", errors.New("Realtime Workspace ID 不能为空")
		}
		region := strings.TrimSpace(cfg.Region)
		if region == "" {
			region = "cn-beijing"
		}
		endpoint = fmt.Sprintf("wss://%s.%s.maas.aliyuncs.com/api-ws/v1/realtime", workspaceID, region)
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New("Realtime Base URL 无效")
	}
	if parsed.Scheme != "wss" && parsed.Scheme != "ws" {
		return "", errors.New("Realtime Base URL 必须使用 ws 或 wss")
	}
	query := parsed.Query()
	query.Set("model", model)
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func buildQwenSessionUpdate(cfg *LiveConfig) map[string]any {
	instructions := cfg.SystemInstruction
	session := map[string]any{
		"modalities":   []string{"text"},
		"instructions": instructions,
		"audio": map[string]any{
			"input": map[string]any{
				"format": map[string]any{"type": "pcm", "sample_rate": 16000},
			},
		},
		"input_audio_transcription": map[string]any{"model": qwenRealtimeTranscriptionModel},
		"turn_detection": map[string]any{
			"type":                cfg.VADType,
			"threshold":           cfg.VADThreshold,
			"silence_duration_ms": cfg.SilenceDurationMs,
		},
		"max_tokens":  cfg.MaxTokens,
		"temperature": cfg.Temperature,
		"top_p":       cfg.TopP,
		"top_k":       cfg.TopK,
	}
	if cfg.RAGEnabled {
		session["instructions"] = instructions + `

知识库规则：每当你判断语音构成完整的面试问题或追问时，必须先调用 search_interview_knowledge 检索候选人的知识库，再生成最终答案。优先使用工具返回的个人经历、项目、技能和参考答案。工具未命中时可以使用通用知识，但禁止编造候选人的公司、项目、职责或数据。不要向用户展示工具调用过程或资料来源。`
		session["tools"] = []map[string]any{{
			"type": "function",
			"function": map[string]any{
				"name":        "search_interview_knowledge",
				"description": "检索候选人的本地面试知识库。回答每个完整的面试问题前都必须调用一次。",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"query": map[string]any{"type": "string", "description": "当前完整面试问题，保留关键技术词和项目名称"},
					},
					"required": []string{"query"},
				},
			},
		}}
	}
	return map[string]any{
		"type":    "session.update",
		"session": session,
	}
}

func buildQwenAudioAppend(data []byte) map[string]any {
	return map[string]any{
		"type":  "input_audio_buffer.append",
		"audio": base64.StdEncoding.EncodeToString(data),
	}
}

// QwenRealtimeSession 封装 Qwen WebSocket。所有写操作均经过 writeMu。
type QwenRealtimeSession struct {
	conn      *websocket.Conn
	writeMu   sync.Mutex
	closeOnce sync.Once
	closed    chan struct{}

	eventMu          sync.Mutex
	completedItems   map[string]struct{}
	completedReplies map[string]struct{}
	toolResponseIDs  map[string]struct{}
}

func (s *QwenRealtimeSession) SendAudio(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	return s.sendJSON(buildQwenAudioAppend(data))
}

func (s *QwenRealtimeSession) sendJSON(value any) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	select {
	case <-s.closed:
		return errors.New("Qwen Realtime 会话已关闭")
	default:
	}
	_ = s.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	return s.conn.WriteJSON(value)
}

func (s *QwenRealtimeSession) Receive() (*LiveMessage, error) {
	for {
		_, data, err := s.conn.ReadMessage()
		if err != nil {
			return nil, err
		}
		var event qwenServerEvent
		if err := json.Unmarshal(data, &event); err != nil {
			continue
		}
		msg := s.convertEvent(event)
		if msg != nil {
			return msg, nil
		}
	}
}

func (s *QwenRealtimeSession) SendToolResponse(toolID string, result string) error {
	if strings.TrimSpace(toolID) == "" {
		return errors.New("Qwen Realtime 工具调用 ID 不能为空")
	}
	return s.sendJSONSequence(
		map[string]any{
			"type": "conversation.item.create",
			"item": map[string]any{
				"type":    "function_call_output",
				"call_id": toolID,
				"output":  result,
			},
		},
		map[string]any{"type": "response.create"},
	)
}

func (s *QwenRealtimeSession) SendToolResponseWithImage(string, []byte, string) error {
	return errors.New("Qwen Realtime 文本会话不支持图片工具响应")
}

func (s *QwenRealtimeSession) Close() error {
	var closeErr error
	s.closeOnce.Do(func() {
		close(s.closed)
		s.writeMu.Lock()
		closeErr = s.conn.Close()
		s.writeMu.Unlock()
	})
	return closeErr
}

type qwenServerEvent struct {
	Type       string `json:"type"`
	ItemID     string `json:"item_id"`
	ResponseID string `json:"response_id"`
	Response   struct {
		ID string `json:"id"`
	} `json:"response"`
	Text       string          `json:"text"`
	Delta      string          `json:"delta"`
	Stash      string          `json:"stash"`
	Transcript string          `json:"transcript"`
	CallID     string          `json:"call_id"`
	Name       string          `json:"name"`
	Arguments  string          `json:"arguments"`
	Error      json.RawMessage `json:"error"`
}

func (s *QwenRealtimeSession) convertEvent(event qwenServerEvent) *LiveMessage {
	switch event.Type {
	case "conversation.item.input_audio_transcription.delta":
		return &LiveMessage{Type: LiveMsgTranscriptPreview, Text: event.Text + event.Stash, ItemID: event.ItemID}
	case "conversation.item.input_audio_transcription.completed":
		if !s.markOnce(s.completedItems, event.ItemID) {
			return nil
		}
		return &LiveMessage{Type: LiveMsgTranscript, Text: event.Transcript, ItemID: event.ItemID}
	case "conversation.item.input_audio_transcription.failed":
		return &LiveMessage{Type: LiveMsgError, Text: qwenEventError(event), ItemID: event.ItemID}
	case "input_audio_buffer.speech_started":
		return &LiveMessage{Type: LiveMsgSpeechStarted, ItemID: event.ItemID}
	case "input_audio_buffer.speech_stopped":
		return &LiveMessage{Type: LiveMsgSpeechStopped, ItemID: event.ItemID}
	case "response.text.delta":
		delta := event.Delta
		if delta == "" {
			delta = event.Text
		}
		return &LiveMessage{Type: LiveMsgAIText, Text: delta, ResponseID: event.ResponseID}
	case "response.text.done":
		// done.text 是完整答案；增量已显示，因此这里不能再次追加。
		return nil
	case "response.function_call_arguments.done":
		if event.Name != "search_interview_knowledge" || strings.TrimSpace(event.CallID) == "" {
			return nil
		}
		s.eventMu.Lock()
		if s.toolResponseIDs == nil {
			s.toolResponseIDs = make(map[string]struct{})
		}
		if event.ResponseID != "" {
			s.toolResponseIDs[event.ResponseID] = struct{}{}
		}
		s.eventMu.Unlock()
		return &LiveMessage{Type: LiveMsgToolCall, ToolName: event.Name, ToolID: event.CallID, Text: event.Arguments, ResponseID: event.ResponseID}
	case "response.done":
		responseID := event.ResponseID
		if responseID == "" {
			responseID = event.Response.ID
		}
		if s.consumeToolResponse(responseID) {
			// Function Call 是中间响应，工具结果回传后还会有最终文字响应。
			return nil
		}
		if !s.markOnce(s.completedReplies, responseID) {
			return nil
		}
		return &LiveMessage{Type: LiveMsgDone, ResponseID: responseID}
	case "error":
		return &LiveMessage{Type: LiveMsgError, Text: qwenEventError(event)}
	default:
		return nil
	}
}

func (s *QwenRealtimeSession) sendJSONSequence(values ...any) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	select {
	case <-s.closed:
		return errors.New("Qwen Realtime 会话已关闭")
	default:
	}
	_ = s.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	for _, value := range values {
		if err := s.conn.WriteJSON(value); err != nil {
			return err
		}
	}
	return nil
}

func (s *QwenRealtimeSession) consumeToolResponse(responseID string) bool {
	if responseID == "" {
		return false
	}
	s.eventMu.Lock()
	defer s.eventMu.Unlock()
	if _, found := s.toolResponseIDs[responseID]; !found {
		return false
	}
	delete(s.toolResponseIDs, responseID)
	return true
}

func (s *QwenRealtimeSession) markOnce(seen map[string]struct{}, id string) bool {
	if id == "" {
		return true
	}
	s.eventMu.Lock()
	defer s.eventMu.Unlock()
	if _, exists := seen[id]; exists {
		return false
	}
	seen[id] = struct{}{}
	return true
}

func qwenEventError(event qwenServerEvent) string {
	if len(event.Error) == 0 || string(event.Error) == "null" {
		if event.Type == "conversation.item.input_audio_transcription.failed" {
			return "语音转录失败"
		}
		return "Qwen Realtime 服务返回错误"
	}
	var detail struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	}
	if json.Unmarshal(event.Error, &detail) == nil {
		if detail.Message != "" {
			return detail.Message
		}
		if detail.Code != "" {
			return detail.Code
		}
	}
	return "Qwen Realtime 服务返回错误"
}
