//go:build windows

package shortcut

import (
	"Q-Solver/pkg/platform"
	"sync"
	"testing"
	"time"
)

type testDelegate struct {
	mu      sync.Mutex
	actions []string
	events  []string
	payload []interface{}
}

func sendTestKeyEvent(t *testing.T, vk byte, keyUp bool) {
	t.Helper()
	if !platform.SendKeyboardInput(uint16(vk), keyUp) {
		t.Skipf("当前 Windows 桌面完整性级别阻止 SendInput，跳过真实按键注入测试: vk=%d keyUp=%v", vk, keyUp)
	}
}

func (d *testDelegate) add(action string) {
	d.mu.Lock()
	d.actions = append(d.actions, action)
	d.mu.Unlock()
}
func (d *testDelegate) TriggerSolve()                  { d.add("solve") }
func (d *testDelegate) TriggerScreenshot()             { d.add("screenshot") }
func (d *testDelegate) TriggerSend()                   { d.add("send") }
func (d *testDelegate) ToggleVisibility()              { d.add("toggle") }
func (d *testDelegate) ToggleClickThrough()            { d.add("clickthrough") }
func (d *testDelegate) MoveWindow(dx, dy int)          { d.add("move") }
func (d *testDelegate) ScrollContent(direction string) { d.add("scroll") }
func (d *testDelegate) EmitEvent(name string, data ...interface{}) {
	d.mu.Lock()
	d.events = append(d.events, name)
	if len(data) > 0 {
		d.payload = append(d.payload, data[0])
	} else {
		d.payload = append(d.payload, nil)
	}
	d.mu.Unlock()
}

func TestAltLeftDispatchesOnceAsynchronously(t *testing.T) {
	m := NewManager()
	m.ReplaceShortcuts(map[string]KeyBinding{"move_left": {ComboID: "37+164"}})
	called := make(chan string, 2)
	m.OnTrigger = func(action string) { called <- action }

	m.mu.Lock()
	m.heldKeys[164] = true
	m.heldKeys[37] = true
	if !onKeysChanged(m) {
		t.Fatal("Alt+Left should be consumed")
	}
	if !onKeysChanged(m) {
		t.Fatal("repeated Alt+Left should remain consumed")
	}
	m.mu.Unlock()

	select {
	case action := <-called:
		if action != "move_left" {
			t.Fatalf("unexpected action: %s", action)
		}
	case <-time.After(time.Second):
		t.Fatal("Alt+Left callback was not dispatched")
	}
	select {
	case action := <-called:
		t.Fatalf("shortcut repeated unexpectedly: %s", action)
	case <-time.After(30 * time.Millisecond):
	}
}

func TestAltLeftCanRepeatAfterArrowRelease(t *testing.T) {
	m := NewManager()
	m.ReplaceShortcuts(map[string]KeyBinding{"move_left": {ComboID: "37+164"}})
	called := make(chan string, 2)
	m.OnTrigger = func(action string) { called <- action }

	m.mu.Lock()
	m.heldKeys[164] = true
	m.heldKeys[37] = true
	onKeysChanged(m)
	releaseKeyLocked(m, 37)
	m.heldKeys[37] = true
	onKeysChanged(m)
	m.mu.Unlock()

	for i := 0; i < 2; i++ {
		select {
		case <-called:
		case <-time.After(time.Second):
			t.Fatalf("expected trigger %d", i+1)
		}
	}
}

func TestCallbackPanicDoesNotStopLaterShortcuts(t *testing.T) {
	m := NewManager()
	m.ReplaceShortcuts(map[string]KeyBinding{
		"move_left":  {ComboID: "37+164"},
		"move_right": {ComboID: "39+164"},
	})
	called := make(chan string, 1)
	m.OnTrigger = func(action string) {
		if action == "move_left" {
			panic("test panic")
		}
		called <- action
	}

	m.mu.Lock()
	m.heldKeys[164] = true
	m.heldKeys[37] = true
	onKeysChanged(m)
	releaseKeyLocked(m, 37)
	m.heldKeys[39] = true
	onKeysChanged(m)
	m.mu.Unlock()

	select {
	case action := <-called:
		if action != "move_right" {
			t.Fatalf("unexpected action: %s", action)
		}
	case <-time.After(time.Second):
		t.Fatal("callback queue did not continue after panic")
	}
}

func TestShortcutMapConcurrentReplacement(t *testing.T) {
	m := NewManager()
	var wg sync.WaitGroup
	for worker := 0; worker < 4; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 500; i++ {
				m.ReplaceShortcuts(map[string]KeyBinding{"move_left": {ComboID: "37+164"}})
				_ = m.GetShortcuts()
			}
		}()
	}
	wg.Wait()
}

func TestAllShortcutActionsDoNotPanic(t *testing.T) {
	d := &testDelegate{}
	s := &Service{delegate: d, manager: NewManager()}
	for _, action := range []string{"screenshot", "send", "solve", "toggle_ui", "toggle", "clickthrough", "move_up", "move_down", "move_left", "move_right", "scroll_up", "scroll_down"} {
		s.handleTrigger(action)
	}
}

func TestManagerRestartReplacesHookLifecycleWithoutLosingConfiguration(t *testing.T) {
	m := NewManager()
	want := map[string]KeyBinding{"toggle": {ComboID: "120", KeyName: "F9"}}
	m.ReplaceShortcuts(want)
	m.Start()
	t.Cleanup(m.Stop)

	m.Restart()
	if globalManager.Load() != m {
		t.Fatal("重启 Hook 后全局快捷键管理器未恢复")
	}
	got := m.GetShortcuts()
	if got["toggle"] != want["toggle"] {
		t.Fatalf("重启 Hook 后快捷键配置丢失: %+v", got)
	}
}

func TestSettingsSubscriptionRestartsHookAndStillReceivesKeys(t *testing.T) {
	var settingsCallback func(map[string]KeyBinding)
	s := NewService(&testDelegate{}, map[string]KeyBinding{
		"probe": {ComboID: "117", KeyName: "F6"},
	}, func(callback func(map[string]KeyBinding)) {
		settingsCallback = callback
	})
	triggered := make(chan string, 2)
	s.manager.OnTrigger = func(action string) { triggered <- action }
	s.Start()
	t.Cleanup(s.Stop)

	pressTestKey := func() {
		sendTestKeyEvent(t, 0x75, false)
		time.Sleep(30 * time.Millisecond)
		sendTestKeyEvent(t, 0x75, true)
	}
	waitForTrigger := func(stage string) {
		select {
		case action := <-triggered:
			if action != "probe" {
				t.Fatalf("%s 收到错误动作: %s", stage, action)
			}
		case <-time.After(time.Second):
			t.Fatalf("%s 快捷键未触发", stage)
		}
	}

	pressTestKey()
	waitForTrigger("保存设置前")
	settingsCallback(map[string]KeyBinding{
		"probe": {ComboID: "117", KeyName: "F6"},
	})
	pressTestKey()
	waitForTrigger("保存设置后")
}

func TestRecordingPreviewDoesNotChangeActiveShortcut(t *testing.T) {
	d := &testDelegate{}
	s := &Service{delegate: d, manager: NewManager()}
	s.manager.ReplaceShortcuts(map[string]KeyBinding{
		"screenshot": {ComboID: "119", KeyName: "F8"},
	})

	s.handleRecord("screenshot", "F1", "112")
	got := s.manager.GetShortcuts()["screenshot"]
	if got.ComboID != "119" || got.KeyName != "F8" {
		t.Fatalf("录制预览提前修改了生效快捷键: %+v", got)
	}
}

func TestRecordingCompletionEmitsSavedEventAndKeepsChangeTemporary(t *testing.T) {
	d := &testDelegate{}
	s := &Service{delegate: d, manager: NewManager()}
	s.manager.ReplaceShortcuts(map[string]KeyBinding{
		"screenshot": {ComboID: "119", KeyName: "F8"},
	})

	s.handleRecordingComplete("screenshot", "F1", "112")
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.events) != 1 || d.events[0] != "shortcut-saved" {
		t.Fatalf("录制完成事件缺失: %+v", d.events)
	}
	if len(d.payload) != 1 || d.payload[0] != "screenshot" {
		t.Fatalf("录制完成事件动作错误: %+v", d.payload)
	}
	got := s.manager.GetShortcuts()["screenshot"]
	if got.ComboID != "119" {
		t.Fatalf("用户保存设置前不应替换生效快捷键: %+v", got)
	}
}

func TestStopRecordingClearsAllCapturedKeyState(t *testing.T) {
	m := NewManager()
	m.recordingKeyFor = "screenshot"
	m.recordedCombo = "112"
	m.maxComboKeys[112] = true
	m.heldKeys[112] = true
	m.triggeredCombo = "112"

	m.StopRecording()
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.recordingKeyFor != "" || m.recordedCombo != "" || len(m.maxComboKeys) != 0 || len(m.heldKeys) != 0 || m.triggeredCombo != "" {
		t.Fatalf("停止录制后仍残留按键状态: recording=%q recorded=%q max=%v held=%v triggered=%q",
			m.recordingKeyFor, m.recordedCombo, m.maxComboKeys, m.heldKeys, m.triggeredCombo)
	}
}

func TestRecordingCallbacksPreservePreviewBeforeCompletion(t *testing.T) {
	m := NewManager()
	events := make(chan string, 2)
	m.OnRecord = func(action, keyName, comboID string) { events <- "preview:" + comboID }
	m.OnRecordingComplete = func(action, keyName, comboID string) { events <- "complete:" + comboID }
	m.StartRecording("screenshot")

	m.mu.Lock()
	m.heldKeys[112] = true
	onKeysChanged(m)
	releaseKeyLocked(m, 112)
	finishRecordingLocked(m)
	m.mu.Unlock()

	want := []string{"preview:112", "complete:112"}
	for index, expected := range want {
		select {
		case got := <-events:
			if got != expected {
				t.Fatalf("回调顺序错误 at %d: got=%q want=%q", index, got, expected)
			}
		case <-time.After(time.Second):
			t.Fatalf("等待回调超时 at %d", index)
		}
	}
}

func TestF1AndF9RemainResponsiveAcrossRepeatedHookRestarts(t *testing.T) {
	d := &testDelegate{}
	var settingsCallback func(map[string]KeyBinding)
	bindings := map[string]KeyBinding{
		"screenshot": {ComboID: "112", KeyName: "F1"},
		"toggle":     {ComboID: "120", KeyName: "F9"},
	}
	s := NewService(d, bindings, func(callback func(map[string]KeyBinding)) {
		settingsCallback = callback
	})
	s.Start()
	t.Cleanup(s.Stop)

	press := func(vk byte) {
		sendTestKeyEvent(t, vk, false)
		time.Sleep(8 * time.Millisecond)
		sendTestKeyEvent(t, vk, true)
		time.Sleep(8 * time.Millisecond)
	}
	const rounds = 20
	for index := 0; index < rounds; index++ {
		press(0x70)
		press(0x78)
		if index == rounds/2 {
			settingsCallback(bindings)
		}
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		d.mu.Lock()
		count := len(d.actions)
		d.mu.Unlock()
		if count == rounds*2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("连续 F1/F9 触发丢失: got=%d want=%d", count, rounds*2)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestRecordingF1CompletesAndF9WorksImmediatelyAfterward(t *testing.T) {
	m := NewManager()
	m.ReplaceShortcuts(map[string]KeyBinding{
		"toggle": {ComboID: "120", KeyName: "F9"},
	})
	recorded := make(chan string, 1)
	triggered := make(chan string, 1)
	m.OnRecordingComplete = func(action, keyName, comboID string) { recorded <- comboID }
	m.OnTrigger = func(action string) { triggered <- action }
	m.Start()
	t.Cleanup(m.Stop)
	m.StartRecording("screenshot")

	sendTestKeyEvent(t, 0x70, false)
	time.Sleep(20 * time.Millisecond)
	sendTestKeyEvent(t, 0x70, true)
	select {
	case comboID := <-recorded:
		if comboID != "112" {
			t.Fatalf("F1 录制结果错误: %s", comboID)
		}
	case <-time.After(time.Second):
		t.Fatal("F1 录制没有结束")
	}

	sendTestKeyEvent(t, 0x78, false)
	time.Sleep(20 * time.Millisecond)
	sendTestKeyEvent(t, 0x78, true)
	select {
	case action := <-triggered:
		if action != "toggle" {
			t.Fatalf("录制后触发错误动作: %s", action)
		}
	case <-time.After(time.Second):
		t.Fatal("F1 录制结束后 F9 仍被吞掉")
	}
}

func TestStaleF1StateIsPrunedBeforeNextShortcut(t *testing.T) {
	m := NewManager()
	m.heldKeys[112] = true // 模拟 Windows 漏掉 F1 KEYUP
	m.heldKeys[164] = true // Alt 仍然真实按住

	pruneReleasedKeysLocked(m, 39, func(vkCode uint32) bool {
		return vkCode == 164
	})
	if m.heldKeys[112] {
		t.Fatal("已松开的 F1 残留状态没有清除")
	}
	if !m.heldKeys[164] {
		t.Fatal("仍按住的 Alt 被错误清除")
	}
}

func TestAllPrimaryShortcutsRemainResponsiveAfterF1AndHookRestart(t *testing.T) {
	d := &testDelegate{}
	bindings := map[string]KeyBinding{
		"screenshot": {ComboID: "112", KeyName: "F1"},
		"send":       {ComboID: "118", KeyName: "F7"},
		"toggle":     {ComboID: "120", KeyName: "F9"},
		"move_down":  {ComboID: "40+164", KeyName: "Alt+↓"},
		"move_left":  {ComboID: "37+164", KeyName: "Alt+←"},
	}
	m := NewManager()
	m.ReplaceShortcuts(bindings)
	m.OnTrigger = d.add
	m.Start()
	t.Cleanup(m.Stop)

	tap := func(vk byte) {
		sendTestKeyEvent(t, vk, false)
		time.Sleep(10 * time.Millisecond)
		sendTestKeyEvent(t, vk, true)
		time.Sleep(10 * time.Millisecond)
	}
	altTap := func(vk byte) {
		sendTestKeyEvent(t, 0xA4, false)
		time.Sleep(10 * time.Millisecond)
		tap(vk)
		sendTestKeyEvent(t, 0xA4, true)
		time.Sleep(10 * time.Millisecond)
	}

	wantRound := []string{"screenshot", "send", "toggle", "move_down", "move_left"}
	const rounds = 10
	for round := 0; round < rounds; round++ {
		tap(0x70)
		tap(0x76)
		tap(0x78)
		altTap(0x28)
		altTap(0x25)
		if round == rounds/2 {
			m.Restart()
		}
	}

	deadline := time.Now().Add(3 * time.Second)
	for {
		d.mu.Lock()
		got := append([]string(nil), d.actions...)
		d.mu.Unlock()
		if len(got) == rounds*len(wantRound) {
			for index, action := range got {
				if expected := wantRound[index%len(wantRound)]; action != expected {
					t.Fatalf("快捷键回调顺序错误 at %d: got=%q want=%q", index, action, expected)
				}
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("F1 后主快捷键发生丢失: got=%d want=%d, actions=%v", len(got), rounds*len(wantRound), got)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
