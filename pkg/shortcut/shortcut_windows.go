//go:build windows

package shortcut

import (
	"Q-Solver/pkg/logger"
	"Q-Solver/pkg/platform"
	"maps"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

type Manager struct {
	mu              sync.Mutex
	callbackQueueMu sync.Mutex
	callbackReady   *sync.Cond
	callbackQueue   []callbackTask
	hHook           uintptr
	hMouseHook      uintptr
	hookThreadID    uint32
	hookRunning     bool
	hookReady       chan struct{}
	hookDone        chan struct{}
	recordingKeyFor string
	maxComboKeys    map[uint32]bool
	heldKeys        map[uint32]bool
	triggeredCombo  string
	recordedCombo   string

	// Callbacks
	OnTrigger           func(action string)
	OnRecord            func(action string, keyName string, comboID string)
	OnRecordingComplete func(action string, keyName string, comboID string)
	OnError             func(msg string)

	// Configuration
	Shortcuts map[string]KeyBinding
}

type callbackTask struct {
	name     string
	callback func()
}

var globalManager atomic.Pointer[Manager]

// Windows callback trampolines cannot be released. Reuse one trampoline per Hook
// for the whole process instead of allocating two more after every settings save.
var (
	keyboardCallback = syscall.NewCallback(keyboardHookProc)
	mouseCallback    = syscall.NewCallback(mouseHookProc)
)

func NewManager() *Manager {
	m := &Manager{
		heldKeys:      make(map[uint32]bool),
		maxComboKeys:  make(map[uint32]bool),
		Shortcuts:     make(map[string]KeyBinding),
		callbackQueue: make([]callbackTask, 0, 16),
	}
	m.callbackReady = sync.NewCond(&m.callbackQueueMu)
	go m.runCallbackQueue()
	return m
}

func (m *Manager) Start() {
	m.mu.Lock()
	if m.hookRunning {
		m.mu.Unlock()
		return
	}
	m.hookRunning = true
	m.hookReady = make(chan struct{})
	m.hookDone = make(chan struct{})
	ready := m.hookReady
	done := m.hookDone
	m.mu.Unlock()

	globalManager.Store(m)
	go m.installHooks(ready, done)
	// 启动完成后再返回，防止设置保存紧接着重启时发生生命周期竞争。
	<-ready
}

func (m *Manager) Stop() {
	globalManager.CompareAndSwap(m, nil)
	m.mu.Lock()
	if !m.hookRunning {
		m.recordingKeyFor = ""
		m.maxComboKeys = make(map[uint32]bool)
		m.heldKeys = make(map[uint32]bool)
		m.triggeredCombo = ""
		m.recordedCombo = ""
		m.mu.Unlock()
		return
	}
	hHook := m.hHook
	hMouseHook := m.hMouseHook
	threadID := m.hookThreadID
	done := m.hookDone
	m.hookRunning = false
	m.hHook = 0
	m.hMouseHook = 0
	m.hookThreadID = 0
	m.recordingKeyFor = ""
	m.maxComboKeys = make(map[uint32]bool)
	m.heldKeys = make(map[uint32]bool)
	m.triggeredCombo = ""
	m.recordedCombo = ""
	m.mu.Unlock()

	if hHook != 0 {
		if platform.UnhookWindowsHookEx(hHook) {
			logger.Println("卸载键盘Hook成功")
		} else {
			logger.Println("卸载键盘Hook失败")
		}
	}
	if hMouseHook != 0 {
		if platform.UnhookWindowsHookEx(hMouseHook) {
			logger.Println("卸载鼠标Hook成功")
		} else {
			logger.Println("卸载鼠标Hook失败")
		}
	}
	if threadID != 0 {
		platform.PostThreadMessage(threadID, platform.WM_QUIT, 0, 0)
	}
	if done != nil {
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			logger.Println("等待快捷键 Hook 线程退出超时")
		}
	}
}

// Restart 在设置保存后重建 Windows Hook。配置保存可能引发 WebView/窗口状态
// 刷新，显式重启可避免系统静默移除低级 Hook 后所有快捷键失效。
func (m *Manager) Restart() {
	m.Stop()
	m.Start()
}

func (m *Manager) StartRecording(action string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.recordingKeyFor = action
	m.maxComboKeys = make(map[uint32]bool)
	m.heldKeys = make(map[uint32]bool)
	m.triggeredCombo = ""
	m.recordedCombo = ""
	logger.Printf("开始录制快捷键: %s\n", action)
}

func (m *Manager) StopRecording() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.recordingKeyFor = ""
	m.maxComboKeys = make(map[uint32]bool)
	m.heldKeys = make(map[uint32]bool)
	m.triggeredCombo = ""
	m.recordedCombo = ""
	logger.Println("停止录制快捷键")
}

func (m *Manager) ReplaceShortcuts(shortcuts map[string]KeyBinding) {
	m.mu.Lock()
	defer m.mu.Unlock()
	clear(m.Shortcuts)
	maps.Copy(m.Shortcuts, shortcuts)
}

func (m *Manager) GetShortcuts() map[string]KeyBinding {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make(map[string]KeyBinding, len(m.Shortcuts))
	maps.Copy(result, m.Shortcuts)
	return result
}

// dispatchCallback keeps callbacks out of the low-level Hook thread. A single FIFO
// worker preserves recording-preview/completion ordering without blocking Windows Hook.
func (m *Manager) dispatchCallback(name string, callback func()) {
	m.callbackQueueMu.Lock()
	m.callbackQueue = append(m.callbackQueue, callbackTask{name: name, callback: callback})
	m.callbackReady.Signal()
	m.callbackQueueMu.Unlock()
}

func (m *Manager) runCallbackQueue() {
	for {
		m.callbackQueueMu.Lock()
		for len(m.callbackQueue) == 0 {
			m.callbackReady.Wait()
		}
		task := m.callbackQueue[0]
		m.callbackQueue[0] = callbackTask{}
		m.callbackQueue = m.callbackQueue[1:]
		m.callbackQueueMu.Unlock()

		func() {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.Printf("快捷键回调异常 (%s): %v\n", task.name, recovered)
				}
			}()
			task.callback()
		}()
	}
}

func releaseKeyLocked(manager *Manager, vkCode uint32) {
	delete(manager.heldKeys, vkCode)
	// A released non-modifier key must allow Alt/Win + key to trigger again
	// while the modifier remains held.
	manager.triggeredCombo = ""
}

// pruneReleasedKeysLocked 修复漏收 KEYUP 后的残留状态。某些 Fn 功能键、
// 截图软件或系统级快捷键可能吞掉松开事件；若不校正，后续所有组合都会
// 永久带上旧键（例如 F1+F9），表现为整套快捷键失效。
func pruneReleasedKeysLocked(manager *Manager, currentVK uint32, isPhysicallyDown func(uint32) bool) {
	pruned := false
	for vkCode := range manager.heldKeys {
		if vkCode == currentVK {
			continue
		}
		if !isPhysicallyDown(vkCode) {
			delete(manager.heldKeys, vkCode)
			pruned = true
		}
	}
	if pruned {
		manager.triggeredCombo = ""
	}
}

func (m *Manager) installHooks(ready, done chan struct{}) {
	// Windows 低级 Hook 必须由安装它的同一个系统线程持续运行消息循环。
	// Go 协程可能在系统线程之间迁移，因此必须显式锁定，否则 Hook 会随机失效。
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer close(done)

	threadID := platform.GetCurrentThreadID()

	// 获取模块句柄
	hMod := platform.GetModuleHandle("")

	// 安装键盘钩子
	hHook := platform.SetWindowsHookEx(platform.WH_KEYBOARD_LL, keyboardCallback, hMod, 0)
	if hHook == 0 {
		logger.Println("安装键盘钩子失败")
	}

	// 安装鼠标钩子
	hMouseHook := platform.SetWindowsHookEx(platform.WH_MOUSE_LL, mouseCallback, hMod, 0)
	if hMouseHook == 0 {
		logger.Println("安装鼠标钩子失败")
	}

	if hHook == 0 && hMouseHook == 0 {
		m.mu.Lock()
		m.hookRunning = false
		m.mu.Unlock()
		close(ready)
		return
	}

	m.mu.Lock()
	if globalManager.Load() != m || !m.hookRunning {
		m.mu.Unlock()
		if hHook != 0 {
			platform.UnhookWindowsHookEx(hHook)
		}
		if hMouseHook != 0 {
			platform.UnhookWindowsHookEx(hMouseHook)
		}
		close(ready)
		return
	}
	m.hHook = hHook
	m.hMouseHook = hMouseHook
	m.hookThreadID = threadID
	m.mu.Unlock()
	close(ready)

	// 消息循环
	var msg platform.MSG
	for platform.GetMessage(&msg, 0, 0, 0) > 0 {
		// 保持线程活跃以处理钩子消息
	}

	m.mu.Lock()
	if m.hookThreadID == threadID {
		m.hHook = 0
		m.hMouseHook = 0
		m.hookThreadID = 0
	}
	m.mu.Unlock()
}

// 这里解释了为什么只能吞掉第二个键，所以导致丢失焦点的问题：（其实是因为alt键的问题）
// 第一个键（Alt）按下：
// 记录：heldKeys = {Alt}
// 判断：有快捷键是只按 Alt 的吗？ -> 没有。
// 结果：放行（Chrome 收到 Alt）。
// 第二个键（~）按下：
// 记录：heldKeys = {Alt, ~}
// 判断：有快捷键是 Alt + ~ 的吗？ -> 有！
// 结果：拦截（return 1，Chrome 收不到 ~）。
func keyboardHookProc(nCode int, wParam uintptr, lParam uintptr) uintptr {
	manager := globalManager.Load()
	if manager == nil {
		return 0
	}
	// 只有当 nCode >= 0 时才处理消息，否则直接放行
	if nCode >= 0 {
		// 将 lParam 指针转换为键盘钩子结构体
		kbd := (*platform.KBDLLHOOKSTRUCT)(unsafe.Pointer(lParam))
		// 监听按下事件 (WM_KEYDOWN) 或 系统按键按下 (WM_SYSKEYDOWN，比如按住 Alt 时)
		if wParam == platform.WM_KEYDOWN || wParam == platform.WM_SYSKEYDOWN {
			manager.mu.Lock()
			pruneReleasedKeysLocked(manager, kbd.VkCode, func(vkCode uint32) bool {
				return platform.GetAsyncKeyState(int(vkCode))&0x8000 != 0
			})
			manager.heldKeys[kbd.VkCode] = true
			consumed := onKeysChanged(manager)
			manager.mu.Unlock()
			if consumed {
				return 1
			}
		}
		// 处理松开事件
		if wParam == platform.WM_KEYUP || wParam == platform.WM_SYSKEYUP {
			manager.mu.Lock()
			// 1. 从 map 中移除该键
			releaseKeyLocked(manager, kbd.VkCode)

			// 录制模式下，松开按键也要检查是否结束录制
			if manager.recordingKeyFor != "" {
				if len(manager.heldKeys) == 0 {
					finishRecordingLocked(manager)
				}
				manager.mu.Unlock()
				return 1 // 录制期间吞掉所有按键
			}
			manager.mu.Unlock()
		}
	}

	// 如果不是我们要拦截的键，或者 nCode < 0，必须调用 CallNextHookEx
	// 否则会导致系统键盘卡死或其他人无法使用键盘
	return platform.CallNextHookEx(0, nCode, wParam, lParam)
}

func mouseHookProc(nCode int, wParam uintptr, lParam uintptr) uintptr {
	manager := globalManager.Load()
	if manager == nil {
		return 0
	}
	if nCode >= 0 {
		mouseStruct := (*platform.MSLLHOOKSTRUCT)(unsafe.Pointer(lParam))
		var vkCode uint32
		isDown := false
		isUp := false

		switch wParam {
		case platform.WM_XBUTTONDOWN:
			isDown = true
			xButton := (mouseStruct.MouseData >> 16) & 0xFFFF
			switch xButton {
			case 1:
				vkCode = platform.VK_XBUTTON1
			case 2:
				vkCode = platform.VK_XBUTTON2
			}
		case platform.WM_XBUTTONUP:
			isUp = true
			xButton := (mouseStruct.MouseData >> 16) & 0xFFFF
			switch xButton {
			case 1:
				vkCode = platform.VK_XBUTTON1
			case 2:
				vkCode = platform.VK_XBUTTON2
			}
		}

		if vkCode != 0 {
			if isDown {
				manager.mu.Lock()
				pruneReleasedKeysLocked(manager, vkCode, func(key uint32) bool {
					return platform.GetAsyncKeyState(int(key))&0x8000 != 0
				})
				manager.heldKeys[vkCode] = true
				consumed := onKeysChanged(manager)
				manager.mu.Unlock()
				if consumed {
					return 1
				}
			} else if isUp {
				manager.mu.Lock()
				releaseKeyLocked(manager, vkCode)
				// 录制模式下，松开按键也要检查是否结束录制
				if manager.recordingKeyFor != "" {
					if len(manager.heldKeys) == 0 {
						finishRecordingLocked(manager)
					}
					manager.mu.Unlock()
					return 1
				}
				manager.mu.Unlock()
			}
		}
	}
	return platform.CallNextHookEx(0, nCode, wParam, lParam)
}

func onKeysChanged(manager *Manager) bool {
	if manager == nil {
		return false
	}

	// --- 录制模式 ---
	if manager.recordingKeyFor != "" {
		// 更新最大按键组合
		if len(manager.heldKeys) >= len(manager.maxComboKeys) {
			manager.maxComboKeys = make(map[uint32]bool)
			for k, v := range manager.heldKeys {
				manager.maxComboKeys[k] = v
			}
		}

		// 实时发给前端显示
		readableName := GetReadableName(manager.maxComboKeys)
		comboID := GetComboID(manager.maxComboKeys)
		if manager.OnRecord != nil && comboID != manager.recordedCombo {
			callback := manager.OnRecord
			action := manager.recordingKeyFor
			manager.recordedCombo = comboID
			manager.dispatchCallback("record", func() {
				callback(action, readableName, comboID)
			})
		}
		return true // 吞掉按键
	}

	// --- 正常模式 ---
	// 将当前按下的所有键生成 ID，去配置里查
	currentComboID := GetComboID(manager.heldKeys)
	for action, savedComboID := range manager.Shortcuts {
		if savedComboID.ComboID == currentComboID {
			if manager.triggeredCombo == currentComboID {
				return true
			}
			manager.triggeredCombo = currentComboID
			if manager.OnTrigger != nil {
				callback := manager.OnTrigger
				// 低级 Hook 线程只负责拦截按键；窗口、Wails 和网络操作异步执行。
				manager.dispatchCallback(action, func() { callback(action) })
			}
			return true // 吞掉按键
		}
	}
	return false
}

// finishRecordingLocked requires manager.mu to be held by the caller.
func finishRecordingLocked(manager *Manager) {
	if manager == nil || manager.recordingKeyFor == "" {
		return
	}

	// 如果没有按任何键（比如直接点击录制然后点别的），忽略
	if len(manager.maxComboKeys) == 0 {
		manager.recordingKeyFor = ""
		return
	}

	comboID := GetComboID(manager.maxComboKeys)
	readableName := GetReadableName(manager.maxComboKeys)
	action := manager.recordingKeyFor

	// 退出录制模式
	manager.recordingKeyFor = ""
	manager.maxComboKeys = nil
	manager.recordedCombo = ""

	// 异步调用回调，避免阻塞 Hook 线程
	if manager.OnRecordingComplete != nil {
		callback := manager.OnRecordingComplete
		manager.dispatchCallback("recording-complete", func() {
			callback(action, readableName, comboID)
		})
	}
}
