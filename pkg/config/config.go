package config

import (
	"Q-Solver/pkg/shortcut"
	"encoding/json"
	"runtime"
)

type Config struct {
	APIKey             string                         `json:"apiKey,omitempty"`
	Provider           string                         `json:"provider,omitempty"`
	Model              string                         `json:"model,omitempty"`
	BaseURL            string                         `json:"baseURL,omitempty"`
	Prompt             string                         `json:"prompt,omitempty"`
	Opacity            float64                        `json:"opacity,omitempty"`
	NoCompression      bool                           `json:"noCompression,omitempty"`
	CompressionQuality int                            `json:"compressionQuality,omitempty"`
	Sharpening         float64                        `json:"sharpening,omitempty"`
	Grayscale          bool                           `json:"grayscale,omitempty"`
	KeepContext        bool                           `json:"keepContext,omitempty"`
	InterruptThinking  bool                           `json:"interruptThinking,omitempty"`
	ScreenshotMode     string                         `json:"screenshotMode,omitempty"`
	ResumePath         string                         `json:"resumePath,omitempty"`
	ResumeBase64       string                         `json:"-"`
	ResumeContent      string                         `json:"resumeContent,omitempty"`
	UseMarkdownResume  bool                           `json:"useMarkdownResume,omitempty"`
	Shortcuts          map[string]shortcut.KeyBinding `json:"shortcuts,omitempty"`

	// LLM 生成参数
	Temperature    float64 `json:"temperature,omitempty"`
	TopP           float64 `json:"topP,omitempty"`
	TopK           int     `json:"topK,omitempty"`
	MaxTokens      int     `json:"maxTokens,omitempty"`
	ThinkingBudget int     `json:"thinkingBudget,omitempty"`

	// 辅助模型（用于总结对话生成问题导图）
	AssistantModel string `json:"assistantModel,omitempty"`

	// Live API
	UseLiveApi bool `json:"useLiveApi,omitempty"`

	// 语音转文字（OpenAI-compatible /audio/transcriptions）
	STTEnabled  bool   `json:"sttEnabled,omitempty"`
	STTAPIKey   string `json:"sttAPIKey,omitempty"`
	STTBaseURL  string `json:"sttBaseURL,omitempty"`
	STTModel    string `json:"sttModel,omitempty"`
	STTLanguage string `json:"sttLanguage,omitempty"`
	VoiceReply  bool   `json:"voiceReply,omitempty"`

	// 窗口尺寸
	WindowWidth  int  `json:"windowWidth,omitempty"`
	WindowHeight int  `json:"windowHeight,omitempty"`
	AIFontSize   int  `json:"aiFontSize,omitempty"`
	CodeWrap     bool `json:"codeWrap,omitempty"`
}

const DefaultModel = "gemini-2.5-flash"

const DefaultPrompt = `你是一名严谨、高效的通用解题助手。请准确识别用户提供的题目（包括截图中的文字、代码、公式和选项），直接围绕题目作答。

回答规则：
1. 先判断题目是否属于编程/算法题。
2. 如果是编程或算法题：
   - 先用“思路”简要说明核心方法、关键步骤和必要的边界情况，不要冗长展开。
   - 再用“代码实现”给出完整、可运行的代码，禁止只给伪代码或零散片段。
   - 优先使用题目指定的编程语言；题目未指定时默认使用 Python。
   - 保留题目要求的类名、函数名、输入输出格式，并补齐必要的 import、类型和入口代码。
   - 最后简要给出时间复杂度和空间复杂度。
3. 如果不是编程题：
   - 先用“原因”或“解析”简要说明判断依据、计算过程或关键知识点。
   - 最后用“答案”明确给出最终结论；选择题同时写出选项编号和选项内容。
4. 对数学题保留必要推导和单位；对判断题说明关键依据；对简答题突出得分点。
5. 如果题目信息不完整或截图不清晰，先指出缺失信息，再基于可见内容给出最合理的假设和答案，不要编造题目条件。
6. 默认使用简体中文回答，表达清晰、精炼，避免与解题无关的客套话。`

func NewDefaultConfig() Config {
	return Config{
		APIKey:             "",
		Model:              DefaultModel,
		BaseURL:            "",
		ResumePath:         "",
		Prompt:             DefaultPrompt,
		Opacity:            1.0,
		KeepContext:        false,
		InterruptThinking:  false,
		ScreenshotMode:     "window",
		NoCompression:      false,
		CompressionQuality: 80,
		Sharpening:         0.0,
		Grayscale:          false,
		UseMarkdownResume:  false,
		ResumeBase64:       "",
		ResumeContent:      "",
		Provider:           "google",

		Shortcuts: getDefaultShortcuts(),

		// LLM 生成参数默认值
		Temperature:    1.0,
		TopP:           0.95,
		TopK:           40,
		MaxTokens:      8192,
		ThinkingBudget: 16000,

		// 辅助模型
		AssistantModel: "",

		// Live API
		UseLiveApi:  false,
		STTEnabled:  false,
		STTModel:    "whisper-1",
		STTLanguage: "zh",
		VoiceReply:  true,

		// 窗口尺寸默认值
		WindowWidth:  0,
		WindowHeight: 0,
		AIFontSize:   14,
		CodeWrap:     false,
	}
}

// getDefaultShortcuts 根据平台返回默认快捷键配置
func getDefaultShortcuts() map[string]shortcut.KeyBinding {
	if runtime.GOOS == "darwin" {
		// macOS 使用简化的快捷键（不依赖 Windows VK 码）
		return map[string]shortcut.KeyBinding{
			"screenshot":   {ComboID: "Cmd+1", KeyName: "⌘1"},
			"send":         {ComboID: "Cmd+2", KeyName: "⌘2"},
			"toggle":       {ComboID: "Cmd+3", KeyName: "⌘3"},
			"clickthrough": {ComboID: "Cmd+4", KeyName: "⌘4"},
			"move_up":      {ComboID: "Cmd+Option+Up", KeyName: "⌘⌥↑"},
			"move_down":    {ComboID: "Cmd+Option+Down", KeyName: "⌘⌥↓"},
			"move_left":    {ComboID: "Cmd+Option+Left", KeyName: "⌘⌥←"},
			"move_right":   {ComboID: "Cmd+Option+Right", KeyName: "⌘⌥→"},
			"scroll_up":    {ComboID: "Cmd+Option+Shift+Up", KeyName: "⌘⌥⇧↑"},
			"scroll_down":  {ComboID: "Cmd+Option+Shift+Down", KeyName: "⌘⌥⇧↓"},
		}
	}
	// Windows 默认快捷键
	return map[string]shortcut.KeyBinding{
		"screenshot":   {ComboID: "119", KeyName: "F8"},
		"send":         {ComboID: "118", KeyName: "F7"},
		"toggle":       {ComboID: "120", KeyName: "F9"},
		"clickthrough": {ComboID: "121", KeyName: "F10"},
		"move_up":      {ComboID: "38+164", KeyName: "Alt+↑"},
		"move_down":    {ComboID: "40+164", KeyName: "Alt+↓"},
		"move_left":    {ComboID: "37+164", KeyName: "Alt+←"},
		"move_right":   {ComboID: "39+164", KeyName: "Alt+→"},
		"scroll_up":    {ComboID: "33+164", KeyName: "Alt+PgUp"},
		"scroll_down":  {ComboID: "34+164", KeyName: "Alt+PgDn"},
	}
}

func (c *Config) ToJSON() string {
	data, _ := json.MarshalIndent(c, "", "  ")
	return string(data)
}

func (c *Config) Validate() error {
	if c.ScreenshotMode != "" && c.ScreenshotMode != "fullscreen" && c.ScreenshotMode != "window" {
		return &ValidationError{Field: "screenshotMode", Message: "截图模式必须是 'fullscreen' 或 'window'"}
	}
	if c.Opacity < 0 || c.Opacity > 1 {
		return &ValidationError{Field: "opacity", Message: "透明度必须在 0-1 之间"}
	}
	if c.CompressionQuality < 1 || c.CompressionQuality > 100 {
		return &ValidationError{Field: "compressionQuality", Message: "压缩质量必须在 1-100 之间"}
	}
	if c.AIFontSize < 10 || c.AIFontSize > 32 {
		return &ValidationError{Field: "aiFontSize", Message: "AI 字体大小必须在 10-32 之间"}
	}
	return nil
}

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}
