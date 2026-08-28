package main

import (
	"Q-Solver/pkg/config"
	"Q-Solver/pkg/live"
	"Q-Solver/pkg/llm"
	"Q-Solver/pkg/logger"
	"Q-Solver/pkg/platform"
	"Q-Solver/pkg/resume"
	"Q-Solver/pkg/screen"
	"Q-Solver/pkg/shortcut"
	"Q-Solver/pkg/solution"
	"Q-Solver/pkg/state"
	"Q-Solver/pkg/stt"
	"Q-Solver/pkg/task"
	"Q-Solver/pkg/tools"
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// 这个只作为连接前端和后端的中间层
type App struct {
	ctx context.Context

	// 管理器
	configManager *config.ConfigManager
	stateManager  *state.StateManager
	taskManager   *task.TaskCoordinator

	// 业务服务
	llmService      *llm.Service
	resumeService   *resume.Service
	shortcutService *shortcut.Service
	screenService   *screen.Service
	solver          *solution.Solver
	liveManager     *live.LiveSessionManager
	toolRegistry    *tools.Registry

	// 手动截图缓存（发布版兼容：最多 3 张）
	screenshotMu     sync.Mutex
	screenshotBuffer []string
}

// NewApp 创建 App 实例
func NewApp() *App {
	configManager := config.NewConfigManager()

	app := &App{
		configManager: configManager,
		stateManager:  state.NewStateManager(),
		taskManager:   task.NewTaskCoordinator(),
		screenService: screen.NewService(),
		toolRegistry:  tools.DefaultRegistry,
	}

	return app
}

// GetRegisteredTools 返回已装载到最终程序中的工具名称。
func (a *App) GetRegisteredTools() []string {
	return a.toolRegistry.List()
}

// CaptureExamQuestion 执行发布版的截图工具并返回 data URL。
func (a *App) CaptureExamQuestion() (string, error) {
	result := a.toolRegistry.Execute(&tools.ToolContext{Ctx: a.ctx}, "manual", "get_exam_question")
	if result.Error != nil {
		return "", result.Error
	}
	return "data:" + result.ImageMimeType + ";base64," + base64.StdEncoding.EncodeToString(result.ImageData), nil
}

// Startup Wails 启动回调
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	// 加载配置
	if err := a.configManager.Load(); err != nil {
		logger.Printf("加载配置失败: %v", err)
	}

	// 应用保存的窗口尺寸
	cfg := a.configManager.Get()
	if cfg.WindowWidth > 0 && cfg.WindowHeight > 0 {
		runtime.WindowSetSize(ctx, cfg.WindowWidth, cfg.WindowHeight)
		logger.Printf("应用保存的窗口尺寸: %dx%d", cfg.WindowWidth, cfg.WindowHeight)
	}

	// 初始化状态管理器
	a.stateManager.Startup(ctx, a.EmitEvent)

	// 初始化屏幕服务
	a.screenService.Startup(ctx)

	// 初始化 LLM 服务
	a.llmService = llm.NewService(a.configManager.Get(), a.configManager)
	a.solver = solution.NewSolver(a.llmService.GetProvider())

	// 初始化简历服务
	a.resumeService = resume.NewService(a.configManager.Get(), a.configManager)

	// 初始化快捷键服务
	a.shortcutService = shortcut.NewService(a, a.configManager.Get().Shortcuts, func(callback func(map[string]shortcut.KeyBinding)) {
		a.configManager.Subscribe(func(NewConfig config.Config, oldConfig config.Config) {
			callback(NewConfig.Shortcuts)
		})
	})
	a.shortcutService.Start()
	logger.Println("快捷键服务已初始化")

	// 订阅配置变更 - 用于 solver 和其他特殊逻辑
	a.configManager.Subscribe(a.onConfigChanged)
	logger.Println("配置变更订阅已注册")

	// 初始化 Live Session 管理器
	a.liveManager = live.NewLiveSessionManager(
		ctx,
		a.llmService,
		a.configManager,
		a.screenService,
		a.EmitEvent,
	)

	// 直接设置为就绪状态
	a.stateManager.UpdateInitStatus(state.StatusReady)
}

// onConfigChanged 配置变更回调 - 仅处理需要 app 层处理的逻辑
func (a *App) onConfigChanged(NewConfig config.Config, oldConfig config.Config) {
	// 更新 solver 的 provider
	if a.solver != nil {
		a.solver.SetProvider(a.llmService.GetProvider())
	}

	// 如果关闭了上下文，清空历史
	if !NewConfig.KeepContext && a.solver != nil {
		a.solver.ClearHistory()
	}

	if NewConfig.UseLiveApi != oldConfig.UseLiveApi {
		if NewConfig.UseLiveApi == true && a.liveManager.IsActive() {
			logger.Println("配置变更，重连 Live Session...")
			a.StopLiveSession()
			if err := a.StartLiveSession(); err != nil {
				logger.Printf("Live Session 重连失败: %v", err)
			}
		}
		if NewConfig.UseLiveApi == false && a.liveManager.IsActive() {
			a.StopLiveSession()
		}
	}

	logger.Println("配置已更新并应用")
}

// OnShutdown Wails 关闭回调
func (a *App) OnShutdown(ctx context.Context) {
	if a.shortcutService != nil {
		a.shortcutService.Stop()
	}
	// 保存配置
	if err := a.configManager.Save(); err != nil {
		logger.Printf("保存配置失败: %v", err)
	}
}

// ==================== 事件与状态 ====================

// EmitEvent 发送事件到前端
func (a *App) EmitEvent(eventName string, data ...interface{}) {
	runtime.EventsEmit(a.ctx, eventName, data...)
}

// GetInitStatus 获取初始化状态
func (a *App) GetInitStatus() string {
	return a.stateManager.GetInitStatusString()
}

// ==================== 配置管理 ====================

// GetSettings 返回当前配置
func (a *App) GetSettings() config.Config {
	return a.configManager.Get()
}

// UpdateSettings 更新配置（从前端 JSON）
func (a *App) UpdateSettings(configJson string) string {
	if err := a.configManager.UpdateFromJSON(configJson); err != nil {
		return err.Error()
	}
	return ""
}

// SyncSettingsToDefaultSettings 兼容旧接口
// Deprecated: 使用 UpdateSettings 替代
func (a *App) SyncSettingsToDefaultSettings(configJson string) string {
	return a.UpdateSettings(configJson)
}

// ==================== 窗口控制 ====================

// ToggleVisibility 切换可见性
func (a *App) ToggleVisibility() {
	a.stateManager.ToggleVisibility()
}

// ToggleClickThrough 切换鼠标穿透
func (a *App) ToggleClickThrough() {
	a.stateManager.ToggleClickThrough()
}

// MoveWindow 移动窗口
func (a *App) MoveWindow(dx, dy int) {
	a.stateManager.MoveWindow(dx, dy)
}

// RestoreFocus 恢复焦点
func (a *App) RestoreFocus() {
	a.stateManager.RestoreFocus()
}

// RemoveFocus 移除焦点
func (a *App) RemoveFocus() {
	a.stateManager.RemoveFocus()
}

// ==================== 解题相关 ====================

// TriggerSolve 触发解题（快捷键调用）
func (a *App) TriggerSolve() {
	cfg := a.configManager.Get()

	// Live 模式下禁用手动截图
	if cfg.UseLiveApi {
		a.EmitEvent("toast", "当前模式不支持手动截图")
		return
	}

	if cfg.APIKey == "" {
		a.EmitEvent("require-login")
		return
	}

	a.TriggerSend()
}

const maxScreenshots = 3

// TriggerScreenshot 截取并缓存一张图片。
func (a *App) TriggerScreenshot() {
	cfg := a.configManager.Get()
	if cfg.UseLiveApi {
		a.EmitEvent("toast", "当前模式不支持手动截图")
		return
	}
	if cfg.APIKey == "" {
		a.EmitEvent("require-login")
		return
	}
	a.screenshotMu.Lock()
	count := len(a.screenshotBuffer)
	a.screenshotMu.Unlock()
	if count >= maxScreenshots {
		a.EmitEvent("toast", "最多截图 3 张图片，请先发送或删除")
		return
	}
	preview, err := a.GetScreenshotPreview(cfg.CompressionQuality, cfg.Sharpening, cfg.Grayscale, cfg.NoCompression, cfg.ScreenshotMode)
	if err != nil {
		a.EmitEvent("toast", "截图失败: "+err.Error())
		return
	}
	a.screenshotMu.Lock()
	a.screenshotBuffer = append(a.screenshotBuffer, preview.Base64)
	count = len(a.screenshotBuffer)
	a.screenshotMu.Unlock()
	a.EmitEvent("screenshot-taken", preview.Base64, count)
}

// RemoveScreenshot 删除指定索引的缓存截图。
func (a *App) RemoveScreenshot(index int) {
	a.screenshotMu.Lock()
	if index < 0 || index >= len(a.screenshotBuffer) {
		a.screenshotMu.Unlock()
		return
	}
	a.screenshotBuffer = append(a.screenshotBuffer[:index], a.screenshotBuffer[index+1:]...)
	count := len(a.screenshotBuffer)
	a.screenshotMu.Unlock()
	a.EmitEvent("screenshot-removed", index, count)
}

// RemoveLastScreenshot 删除最后一张缓存截图。
func (a *App) RemoveLastScreenshot() {
	a.screenshotMu.Lock()
	if len(a.screenshotBuffer) == 0 {
		a.screenshotMu.Unlock()
		return
	}
	index := len(a.screenshotBuffer) - 1
	a.screenshotBuffer = a.screenshotBuffer[:index]
	count := len(a.screenshotBuffer)
	a.screenshotMu.Unlock()
	a.EmitEvent("screenshot-removed", index, count)
}

// ClearScreenshots 清空缓存截图。
func (a *App) ClearScreenshots() {
	a.screenshotMu.Lock()
	a.screenshotBuffer = nil
	a.screenshotMu.Unlock()
	a.EmitEvent("screenshots-cleared")
}

// TriggerSend 发送缓存截图；无缓存时自动截图后发送。
func (a *App) TriggerSend() {
	cfg := a.configManager.Get()
	if cfg.UseLiveApi {
		a.EmitEvent("toast", "当前模式不支持手动截图")
		return
	}
	if cfg.APIKey == "" {
		a.EmitEvent("require-login")
		return
	}
	if a.taskManager.HasRunningTask() {
		a.EmitEvent("toast", "正在处理中，请稍候...")
		return
	}
	a.screenshotMu.Lock()
	needCapture := len(a.screenshotBuffer) == 0
	a.screenshotMu.Unlock()
	if needCapture {
		a.EmitEvent("toast", "请先使用截图快捷键截取 1-3 张图片")
		return
	}
	a.screenshotMu.Lock()
	screenshots := append([]string(nil), a.screenshotBuffer...)
	a.screenshotBuffer = nil
	a.screenshotMu.Unlock()
	if len(screenshots) == 0 {
		return
	}
	a.EmitEvent("screenshots-cleared")
	a.EmitEvent("start-solving")
	a.EmitEvent("user-message", screenshots[0])
	ctx, taskID := a.taskManager.StartTask("solve")
	go func() { defer a.taskManager.CompleteTask(taskID); a.solveInternal(ctx, screenshots) }()
}

// solveInternal 内部解题逻辑
func (a *App) solveInternal(ctx context.Context, screenshots []string) bool {
	cfg := a.configManager.Get()

	if cfg.APIKey == "" {
		a.EmitEvent("require-login")
		return false
	}

	// 读取简历
	resumeBase64, err := a.resumeService.GetResumeBase64()
	if err != nil {
		logger.Printf("读取简历失败: %v\n", err)
	}

	req := solution.Request{
		Config:       cfg,
		Screenshots:  screenshots,
		ResumeBase64: resumeBase64,
	}

	cb := solution.Callbacks{
		EmitEvent: a.EmitEvent,
	}

	return a.solver.Solve(ctx, req, cb)
}

// CancelRunningTask 取消当前运行的任务
func (a *App) CancelRunningTask() bool {
	return a.taskManager.CancelCurrentTask()
}

// IsInterruptThinkingEnabled 是否允许打断思考
func (a *App) IsInterruptThinkingEnabled() bool {
	return a.configManager.Get().InterruptThinking
}

// ==================== 快捷键相关 ====================

// StartRecordingKey 开始录制快捷键
func (a *App) StartRecordingKey(action string) {
	a.shortcutService.StartRecording(action)
}

// StopRecordingKey 停止录制快捷键
func (a *App) StopRecordingKey() {
	if a.shortcutService != nil {
		a.shortcutService.StopRecording()
	}
}

// ScrollContent 滚动内容
func (a *App) ScrollContent(direction string) {
	a.EmitEvent("scroll-content", direction)
}

// CopyCode 复制代码
func (a *App) CopyCode() {
	a.EmitEvent("copy-code")
}

// ==================== 简历相关 ====================

// SelectResume 选择简历文件
func (a *App) SelectResume() string {
	path := a.resumeService.SelectResume(a.ctx)
	if path != "" {
		// 获取配置副本，修改后保存
		cfg := a.configManager.Get()
		cfg.ResumePath = path
		// 序列化并更新配置
		if jsonData, err := json.Marshal(cfg); err == nil {
			_ = a.configManager.UpdateFromJSON(string(jsonData))
		}
	}
	return path
}

// ClearResume 清除简历
func (a *App) ClearResume() {
	a.resumeService.ClearResume()
	// 获取配置副本，修改后保存
	cfg := a.configManager.Get()
	cfg.ResumePath = ""
	cfg.ResumeBase64 = ""
	cfg.ResumeContent = ""
	// 序列化并更新配置
	if jsonData, err := json.Marshal(cfg); err == nil {
		_ = a.configManager.UpdateFromJSON(string(jsonData))
	}
}

// GetResumePDF 获取简历 Base64
func (a *App) GetResumePDF() (string, error) {
	return a.resumeService.GetResumeBase64()
}

// ParseResume 解析简历为 Markdown
func (a *App) ParseResume() (string, error) {
	return a.resumeService.ParseResume(a.ctx, a.llmService.GetProvider())
}

// ==================== 截图相关 ====================

// GetScreenshotPreview 获取截图预览
func (a *App) GetScreenshotPreview(quality int, sharpen float64, grayscale bool, noCompression bool, screenshotMode string) (screen.PreviewResult, error) {
	mode := screenshotMode
	if mode == "" {
		mode = a.configManager.Get().ScreenshotMode
	}
	return a.screenService.CapturePreview(quality, sharpen, grayscale, noCompression, mode)
}

// CheckScreenCapturePermission 检查截图权限 (macOS)
func (a *App) CheckScreenCapturePermission() bool {
	return platform.CheckScreenCaptureAccess()
}

// RequestScreenCapturePermission 请求截图权限 (macOS)
func (a *App) RequestScreenCapturePermission() bool {
	return platform.RequestScreenCaptureAccess()
}

// OpenScreenCaptureSettings 打开系统设置的屏幕录制权限页面 (macOS)
func (a *App) OpenScreenCaptureSettings() {
	platform.OpenScreenCaptureSettings()
}

// CheckMicrophoneAccess 检查麦克风权限状态 (macOS)
// 返回: 0=未决定, 1=已授权, 2=已拒绝
func (a *App) CheckMicrophoneAccess() int {
	return platform.CheckMicrophoneAccess()
}

// RequestMicrophoneAccess 请求麦克风权限 (macOS)
func (a *App) RequestMicrophoneAccess() {
	platform.RequestMicrophoneAccess()
}

// OpenMicrophoneSettings 打开系统设置的麦克风权限页面 (macOS)
func (a *App) OpenMicrophoneSettings() {
	platform.OpenMicrophoneSettings()
}

// SetWindowAlwaysOnTop 设置窗口是否置顶
func (a *App) SetWindowAlwaysOnTop(alwaysOnTop bool) {
	hwnd := a.stateManager.GetHwnd()
	if hwnd == 0 {
		return
	}
	if alwaysOnTop {
		platform.SetWindowLevel(hwnd, platform.WindowLevelFloating)
	} else {
		platform.SetWindowLevel(hwnd, platform.WindowLevelNormal)
	}
}

// ==================== LLM 相关 ====================

// TestConnection 测试模型连通性
// 通过发送一个简单的消息来测试 API 是否可用
func (a *App) TestConnection(apiKey, baseURL, model string) string {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	return a.llmService.TestConnection(ctx, apiKey, baseURL, model)
}

// GetModels 获取模型列表
func (a *App) GetModels(apiKey string, baseURL string) ([]string, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	return a.llmService.GetModels(ctx, apiKey, baseURL)
}

// GetSTTModels 获取语音转文字 API 提供的模型列表。
func (a *App) GetSTTModels(apiKey string, baseURL string) ([]string, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	client := stt.Client{APIKey: apiKey, BaseURL: baseURL}
	return client.GetModels(ctx)
}

// ==================== 导出相关 ====================

// SaveImageToFile 保存图片到文件（弹出文件选择对话框）
func (a *App) SaveImageToFile(base64Data string) (bool, error) {
	// 弹出文件保存对话框
	filename, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "保存图片",
		DefaultFilename: "q-solver-export.png",
		Filters: []runtime.FileFilter{
			{DisplayName: "PNG 图片", Pattern: "*.png"},
		},
	})
	if err != nil {
		return false, err
	}
	if filename == "" {
		return false, nil // 用户取消
	}

	// 解析 base64 数据
	// 移除 data:image/png;base64, 前缀
	const prefix = "data:image/png;base64,"
	data := base64Data
	if len(data) > len(prefix) && data[:len(prefix)] == prefix {
		data = data[len(prefix):]
	}

	// 解码 base64
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return false, err
	}

	// 写入文件
	if err := os.WriteFile(filename, decoded, 0644); err != nil {
		return false, err
	}

	return true, nil
}

// ==================== Live API ====================

// StartLiveSession 启动 Live API 会话
func (a *App) StartLiveSession() error {
	cfg := a.configManager.Get()
	if cfg.APIKey == "" {
		a.EmitEvent("require-login")
		return nil
	}
	return a.liveManager.Start()
}

// StopLiveSession 停止 Live API 会话
func (a *App) StopLiveSession() {
	a.liveManager.Stop()
}
