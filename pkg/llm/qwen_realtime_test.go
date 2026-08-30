package llm

import (
	"Q-Solver/pkg/config"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func testRealtimeConfig() *LiveConfig {
	return &LiveConfig{
		Model:             "qwen3.5-omni-plus-realtime",
		SystemInstruction: "voice-only-prompt",
		APIKey:            "secret",
		WorkspaceID:       "workspace-123",
		Region:            "cn-beijing",
		Temperature:       0.3,
		TopP:              0.8,
		TopK:              0,
		MaxTokens:         2000,
		VADType:           "semantic_vad",
		VADThreshold:      0.5,
		SilenceDurationMs: 800,
	}
}

func TestQwenRealtimeWebSocketHandshakeSessionUpdateAndAudio(t *testing.T) {
	payloads := make(chan map[string]any, 1)
	serverErrors := make(chan error, 1)
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			serverErrors <- fmt.Errorf("Authorization 请求头不正确")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if r.URL.Query().Get("model") != "qwen3.5-omni-plus-realtime" {
			serverErrors <- fmt.Errorf("model query 不正确")
			http.Error(w, "bad model", http.StatusBadRequest)
			return
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			serverErrors <- err
			return
		}
		defer conn.Close()
		var update map[string]any
		if err := conn.ReadJSON(&update); err != nil {
			serverErrors <- err
			return
		}
		if update["type"] != "session.update" {
			serverErrors <- fmt.Errorf("首个事件不是 session.update: %v", update["type"])
			return
		}
		if err := conn.WriteJSON(map[string]any{"type": "session.updated"}); err != nil {
			serverErrors <- err
			return
		}
		var audioPayload map[string]any
		if err := conn.ReadJSON(&audioPayload); err != nil {
			serverErrors <- err
			return
		}
		payloads <- audioPayload
	}))
	defer server.Close()

	cfg := testRealtimeConfig()
	cfg.BaseURL = "ws" + strings.TrimPrefix(server.URL, "http")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	session, err := NewQwenRealtimeProvider().ConnectLive(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	pcm := []byte{10, 20, 30, 40}
	if err := session.SendAudio(pcm); err != nil {
		t.Fatal(err)
	}

	select {
	case err := <-serverErrors:
		t.Fatal(err)
	case payload := <-payloads:
		if payload["type"] != "input_audio_buffer.append" {
			t.Fatalf("音频事件类型错误: %v", payload)
		}
		decoded, err := base64.StdEncoding.DecodeString(payload["audio"].(string))
		if err != nil || string(decoded) != string(pcm) {
			t.Fatalf("WebSocket 音频内容错误: decoded=%v err=%v", decoded, err)
		}
	case <-ctx.Done():
		t.Fatal("等待模拟服务端音频事件超时")
	}
}

// TestQwenRealtimeIntegration 是显式启用的真实服务握手测试。默认跳过，
// 凭据只从当前进程环境变量读取，不写入源码、配置或测试日志。
func TestQwenRealtimeIntegration(t *testing.T) {
	apiKey := os.Getenv("QWEN_REALTIME_API_KEY")
	workspaceID := os.Getenv("QWEN_REALTIME_WORKSPACE_ID")
	if apiKey == "" || workspaceID == "" {
		t.Skip("未提供 Qwen Realtime 集成测试环境变量")
	}
	cfg := testRealtimeConfig()
	cfg.APIKey = apiKey
	cfg.WorkspaceID = workspaceID
	cfg.SystemInstruction = config.DefaultRealtimePrompt
	cfg.Temperature = 0.4
	cfg.TopP = 0.8
	cfg.TopK = 20
	cfg.MaxTokens = 600
	ragContext := os.Getenv("QWEN_REALTIME_RAG_CONTEXT")
	cfg.RAGEnabled = ragContext != ""
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	session, err := NewQwenRealtimeProvider().ConnectLive(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
}

// TestQwenRealtimeAudioIntegration 发送真实的 16kHz/S16/单声道 PCM，验证
// 转录、semantic VAD 自动结束和文字回答。默认跳过。
func TestQwenRealtimeAudioIntegration(t *testing.T) {
	apiKey := os.Getenv("QWEN_REALTIME_API_KEY")
	workspaceID := os.Getenv("QWEN_REALTIME_WORKSPACE_ID")
	wavPath := os.Getenv("QWEN_REALTIME_WAV_PATH")
	if apiKey == "" || workspaceID == "" || wavPath == "" {
		t.Skip("未提供 Qwen Realtime 音频集成测试环境变量")
	}
	pcm, err := readPCM16Mono16kWAV(wavPath)
	if err != nil {
		t.Fatal(err)
	}
	cfg := testRealtimeConfig()
	cfg.APIKey = apiKey
	cfg.WorkspaceID = workspaceID
	cfg.SystemInstruction = config.DefaultRealtimePrompt
	cfg.Temperature = 0.4
	cfg.TopP = 0.8
	cfg.TopK = 20
	cfg.MaxTokens = 600
	ragContext := os.Getenv("QWEN_REALTIME_RAG_CONTEXT")
	cfg.RAGEnabled = ragContext != ""
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	session, err := NewQwenRealtimeProvider().ConnectLive(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	messages := make(chan *LiveMessage, 128)
	receiveErrors := make(chan error, 1)
	go func() {
		for {
			message, receiveErr := session.Receive()
			if receiveErr != nil {
				receiveErrors <- receiveErr
				return
			}
			messages <- message
		}
	}()

	const packetSize = 16000 * 2 * 30 / 1000
	for offset := 0; offset < len(pcm); offset += packetSize {
		end := offset + packetSize
		if end > len(pcm) {
			end = len(pcm)
		}
		if err := session.SendAudio(pcm[offset:end]); err != nil {
			t.Fatal(err)
		}
		time.Sleep(30 * time.Millisecond)
	}
	// 保持纯 append，发送 1.5 秒静音供服务端 semantic VAD 判定说话结束。
	silencePacket := make([]byte, packetSize)
	for i := 0; i < 50; i++ {
		if err := session.SendAudio(silencePacket); err != nil {
			t.Fatal(err)
		}
		time.Sleep(30 * time.Millisecond)
	}

	var transcript strings.Builder
	var answer strings.Builder
	transcriptCompleted := 0
	responseDone := 0
	toolCalls := 0
	for responseDone == 0 {
		select {
		case <-ctx.Done():
			t.Fatalf("等待回答超时；transcript=%q answer=%q", transcript.String(), answer.String())
		case receiveErr := <-receiveErrors:
			t.Fatalf("接收服务端事件失败: %v", receiveErr)
		case message := <-messages:
			switch message.Type {
			case LiveMsgTranscript:
				transcriptCompleted++
				transcript.Reset()
				transcript.WriteString(message.Text)
			case LiveMsgAIText:
				answer.WriteString(message.Text)
			case LiveMsgToolCall:
				toolCalls++
				if message.ToolName != "search_interview_knowledge" {
					t.Fatalf("收到未知工具调用: %s", message.ToolName)
				}
				if err := session.SendToolResponse(message.ToolID, ragContext); err != nil {
					t.Fatalf("回传 RAG 工具结果失败: %v", err)
				}
			case LiveMsgDone:
				responseDone++
			case LiveMsgError:
				t.Fatalf("服务端返回错误: %s", message.Text)
			}
		}
	}
	if transcriptCompleted != 1 || strings.TrimSpace(transcript.String()) == "" {
		t.Fatalf("最终转录次数或内容不正确: count=%d transcript=%q", transcriptCompleted, transcript.String())
	}
	if responseDone != 1 || strings.TrimSpace(answer.String()) == "" {
		t.Fatalf("回答完成次数或内容不正确: count=%d answer=%q", responseDone, answer.String())
	}
	if ragContext != "" && toolCalls != 1 {
		t.Fatalf("RAG 开启时应且只应调用一次知识库工具: count=%d", toolCalls)
	}
	t.Logf("最终转录: %s", transcript.String())
	t.Logf("文字回答: %s", answer.String())
}

func readPCM16Mono16kWAV(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) < 12 || string(data[:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		return nil, fmt.Errorf("不是有效的 WAV 文件")
	}
	validFormat := false
	var pcm []byte
	for offset := 12; offset+8 <= len(data); {
		chunkID := string(data[offset : offset+4])
		chunkSize := int(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
		start := offset + 8
		end := start + chunkSize
		if end > len(data) {
			return nil, fmt.Errorf("WAV chunk 越界")
		}
		switch chunkID {
		case "fmt ":
			if chunkSize >= 16 {
				format := binary.LittleEndian.Uint16(data[start : start+2])
				channels := binary.LittleEndian.Uint16(data[start+2 : start+4])
				sampleRate := binary.LittleEndian.Uint32(data[start+4 : start+8])
				bits := binary.LittleEndian.Uint16(data[start+14 : start+16])
				validFormat = format == 1 && channels == 1 && sampleRate == 16000 && bits == 16
			}
		case "data":
			pcm = append([]byte(nil), data[start:end]...)
		}
		offset = end + chunkSize%2
	}
	if !validFormat {
		return nil, fmt.Errorf("WAV 必须是 16kHz、16-bit、单声道 PCM")
	}
	if len(pcm) == 0 {
		return nil, fmt.Errorf("WAV 不包含 PCM 数据")
	}
	return pcm, nil
}

func TestBuildQwenRealtimeURL(t *testing.T) {
	got, err := buildQwenRealtimeURL(testRealtimeConfig())
	if err != nil {
		t.Fatal(err)
	}
	want := "wss://workspace-123.cn-beijing.maas.aliyuncs.com/api-ws/v1/realtime?model=qwen3.5-omni-plus-realtime"
	if got != want {
		t.Fatalf("URL = %q, want %q", got, want)
	}
}

func TestSessionUpdateUsesRealtimePromptAndParameters(t *testing.T) {
	payload := buildQwenSessionUpdate(testRealtimeConfig())
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{
		`"type":"session.update"`, `"instructions":"voice-only-prompt"`,
		`"modalities":["text"]`, `"sample_rate":16000`,
		`"model":"qwen3-asr-flash-realtime"`, `"type":"semantic_vad"`,
		`"silence_duration_ms":800`, `"temperature":0.3`, `"top_p":0.8`, `"max_tokens":2000`,
	} {
		if !strings.Contains(text, want) {
			t.Errorf("session.update 缺少 %s: %s", want, text)
		}
	}
}

func TestSessionUpdateRegistersRAGToolOnlyWhenEnabled(t *testing.T) {
	cfg := testRealtimeConfig()
	payload := buildQwenSessionUpdate(cfg)
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "search_interview_knowledge") {
		t.Fatal("RAG 关闭时不应注册知识库工具")
	}

	cfg.RAGEnabled = true
	payload = buildQwenSessionUpdate(cfg)
	data, err = json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"search_interview_knowledge", "必须先调用", "禁止编造候选人"} {
		if !strings.Contains(string(data), required) {
			t.Fatalf("RAG session.update 缺少 %q: %s", required, data)
		}
	}
}

func TestQwenFunctionCallMapsToToolAndIntermediateDoneIsSuppressed(t *testing.T) {
	session := &QwenRealtimeSession{
		completedItems:   make(map[string]struct{}),
		completedReplies: make(map[string]struct{}),
		toolResponseIDs:  make(map[string]struct{}),
	}
	tool := session.convertEvent(qwenServerEvent{
		Type:       "response.function_call_arguments.done",
		ResponseID: "tool-response",
		CallID:     "call-1",
		Name:       "search_interview_knowledge",
		Arguments:  `{"query":"介绍一下你的项目"}`,
	})
	if tool == nil || tool.Type != LiveMsgToolCall || tool.ToolID != "call-1" || !strings.Contains(tool.Text, "项目") {
		t.Fatalf("function call mapping failed: %#v", tool)
	}
	if done := session.convertEvent(qwenServerEvent{Type: "response.done", ResponseID: "tool-response"}); done != nil {
		t.Fatalf("工具中间响应不应结束当前轮次: %#v", done)
	}
	final := session.convertEvent(qwenServerEvent{Type: "response.done", ResponseID: "final-response"})
	if final == nil || final.Type != LiveMsgDone {
		t.Fatalf("最终文字响应应结束当前轮次: %#v", final)
	}
}

func TestAudioAppendContainsOnlyAppendAndBase64Audio(t *testing.T) {
	pcm := []byte{0, 1, 2, 253, 254, 255}
	payload := buildQwenAudioAppend(pcm)
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, `"type":"input_audio_buffer.append"`) {
		t.Fatalf("不是 append 事件: %s", text)
	}
	if !strings.Contains(text, base64.StdEncoding.EncodeToString(pcm)) {
		t.Fatalf("音频 Base64 不正确: %s", text)
	}
	if strings.Contains(text, "commit") || strings.Contains(text, "response.create") {
		t.Fatalf("semantic VAD 音频包不应手动提交: %s", text)
	}
	decoded, err := base64.StdEncoding.DecodeString(payload["audio"].(string))
	if err != nil || string(decoded) != string(pcm) {
		t.Fatalf("音频 Base64 不能无损还原 PCM: decoded=%v err=%v", decoded, err)
	}
}

func TestRealtimeLiveConfigDoesNotUseScreenshotSettings(t *testing.T) {
	cfg := config.NewDefaultConfig()
	cfg.APIKey = "screenshot-key"
	cfg.Model = "screenshot-model"
	cfg.Prompt = "screenshot-prompt"
	cfg.RealtimeAPIKey = "voice-key"
	cfg.RealtimeModel = "voice-model"
	cfg.RealtimePrompt = "voice-prompt"

	got := GetRealtimeLiveConfig(cfg)
	if got.APIKey != "voice-key" || got.Model != "voice-model" || got.SystemInstruction != "voice-prompt" {
		t.Fatalf("语音 LiveConfig 未使用独立配置: %+v", got)
	}
	if got.APIKey == cfg.APIKey || got.Model == cfg.Model || got.SystemInstruction == cfg.Prompt {
		t.Fatalf("语音 LiveConfig 污染了截图配置: %+v", got)
	}
}

func TestQwenEventMappingDoesNotDuplicateTranscriptOrAnswer(t *testing.T) {
	s := &QwenRealtimeSession{
		completedItems:   make(map[string]struct{}),
		completedReplies: make(map[string]struct{}),
	}

	preview := s.convertEvent(qwenServerEvent{
		Type: "conversation.item.input_audio_transcription.delta", ItemID: "item-1", Text: "完整", Stash: "预览",
	})
	if preview.Type != LiveMsgTranscriptPreview || preview.Text != "完整预览" {
		t.Fatalf("转录预览映射错误: %#v", preview)
	}

	final := s.convertEvent(qwenServerEvent{
		Type: "conversation.item.input_audio_transcription.completed", ItemID: "item-1", Transcript: "最终问题",
	})
	if final.Type != LiveMsgTranscript || final.Text != "最终问题" {
		t.Fatalf("最终转录映射错误: %#v", final)
	}
	if duplicate := s.convertEvent(qwenServerEvent{
		Type: "conversation.item.input_audio_transcription.completed", ItemID: "item-1", Transcript: "最终问题",
	}); duplicate != nil {
		t.Fatalf("重复 completed 应忽略: %#v", duplicate)
	}

	delta := s.convertEvent(qwenServerEvent{Type: "response.text.delta", ResponseID: "r1", Delta: "增量答案"})
	if delta.Type != LiveMsgAIText || delta.Text != "增量答案" {
		t.Fatalf("回答增量映射错误: %#v", delta)
	}
	if doneText := s.convertEvent(qwenServerEvent{Type: "response.text.done", ResponseID: "r1", Text: "增量答案"}); doneText != nil {
		t.Fatalf("response.text.done 不应重复追加: %#v", doneText)
	}
	done := s.convertEvent(qwenServerEvent{Type: "response.done", ResponseID: "r1"})
	if done.Type != LiveMsgDone {
		t.Fatalf("response.done 映射错误: %#v", done)
	}
	if duplicate := s.convertEvent(qwenServerEvent{Type: "response.done", ResponseID: "r1"}); duplicate != nil {
		t.Fatalf("重复 response.done 应忽略: %#v", duplicate)
	}
	nestedDoneEvent := qwenServerEvent{Type: "response.done"}
	nestedDoneEvent.Response.ID = "r2"
	if nestedDone := s.convertEvent(nestedDoneEvent); nestedDone == nil || nestedDone.ResponseID != "r2" {
		t.Fatalf("嵌套 response.id 映射错误: %#v", nestedDone)
	}
}
