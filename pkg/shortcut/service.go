package shortcut

import (
	"Q-Solver/pkg/logger"
	"fmt"
	"runtime"
)

type Service struct {
	manager  *Manager
	delegate ServiceDelegate
}

// SubscribeFunc 配置变更订阅函数类型
type SubscribeFunc func(callback func(shortcuts map[string]KeyBinding))

func NewService(delegate ServiceDelegate, initialShortcuts map[string]KeyBinding, subscribe SubscribeFunc) *Service {
	s := &Service{
		manager:  NewManager(),
		delegate: delegate,
	}
	s.manager.OnTrigger = s.handleTrigger
	s.manager.OnRecord = s.handleRecord
	s.manager.OnRecordingComplete = s.handleRecordingComplete
	s.manager.OnError = func(msg string) {}

	// 初始化快捷键
	s.manager.ReplaceShortcuts(initialShortcuts)

	// 自注册配置变更回调
	subscribe(func(shortcuts map[string]KeyBinding) {
		s.manager.ReplaceShortcuts(shortcuts)
		s.manager.Restart()
		logger.Println("快捷键配置已更新，Hook 已重启")
	})

	return s
}

func (s *Service) Start() {
	s.manager.Start()
}

func (s *Service) Stop() {
	s.manager.Stop()
}

func (s *Service) GetShortcuts() map[string]KeyBinding {
	return s.manager.GetShortcuts()
}

func (s *Service) StartRecording(action string) {
	// macOS 不支持热键录制
	if runtime.GOOS == "darwin" {
		s.delegate.EmitEvent("toast", "macOS 不支持自定义快捷键，请使用预设快捷键")
		return
	}
	s.manager.StartRecording(action)
}

func (s *Service) StopRecording() {
	s.manager.StopRecording()
}

func (s *Service) handleTrigger(action string) {
	switch action {
	case "screenshot":
		logger.Println("触发截图")
		s.delegate.TriggerScreenshot()
	case "send":
		logger.Println("发送截图并请求回答")
		s.delegate.TriggerSend()
	case "toggle_ui":
		logger.Println("切换界面内容显示状态")
		s.delegate.EmitEvent("toggle-ui-content")
	case "solve":
		logger.Println("触发解题")
		// 兼容旧配置：旧版 solve 快捷键等价于发送截图。
		s.delegate.TriggerSend()
	case "toggle":
		logger.Println("切换可见性")
		s.delegate.ToggleVisibility()
	case "clickthrough":
		logger.Println("切换鼠标穿透")
		s.delegate.ToggleClickThrough()
	case "move_up":
		s.delegate.MoveWindow(0, -10)
	case "move_down":
		s.delegate.MoveWindow(0, 10)
	case "move_left":
		s.delegate.MoveWindow(-10, 0)
	case "move_right":
		s.delegate.MoveWindow(10, 0)
	case "scroll_up":
		s.delegate.ScrollContent("up")
	case "scroll_down":
		s.delegate.ScrollContent("down")
	}
}

func (s *Service) handleRecord(action string, keyName string, comboID string) {
	s.delegate.EmitEvent("key-recorded", map[string]string{
		"action":  action,
		"keyName": keyName,
		"comboID": comboID,
	})
	logger.Printf("快捷键录制预览: %s -> %s->%s\n", action, keyName, comboID)
}

func (s *Service) handleRecordingComplete(action string, keyName string, comboID string) {
	// 检查冲突
	conflict := false
	conflictAction := ""
	for act, binding := range s.manager.GetShortcuts() {
		if binding.ComboID == comboID && act != action {
			conflict = true
			conflictAction = act
			break
		}
	}

	if conflict {
		s.delegate.EmitEvent("shortcut-error", fmt.Sprintf("快捷键冲突：该组合键已被“%s”占用，请重新输入", conflictAction))
		logger.Printf("快捷键录制失败，冲突: %s -> %s\n", action, keyName)
		// 重新开始录制
		s.manager.StartRecording(action)
		return
	}

	// 这里只确认录制结束并通知前端。新按键保留在 tempShortcuts 中，只有用户
	// 点击“保存”后才通过配置订阅替换生效快捷键，取消设置不会污染运行状态。
	s.delegate.EmitEvent("shortcut-saved", action)
}
