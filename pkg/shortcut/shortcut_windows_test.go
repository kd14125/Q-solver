//go:build windows

package shortcut

import (
	"sync"
	"testing"
	"time"
)

type testDelegate struct {
	mu      sync.Mutex
	actions []string
}

func (d *testDelegate) add(action string) {
	d.mu.Lock()
	d.actions = append(d.actions, action)
	d.mu.Unlock()
}
func (d *testDelegate) TriggerSolve()                    { d.add("solve") }
func (d *testDelegate) TriggerScreenshot()               { d.add("screenshot") }
func (d *testDelegate) TriggerSend()                     { d.add("send") }
func (d *testDelegate) ToggleVisibility()                { d.add("toggle") }
func (d *testDelegate) ToggleClickThrough()              { d.add("clickthrough") }
func (d *testDelegate) MoveWindow(dx, dy int)            { d.add("move") }
func (d *testDelegate) ScrollContent(direction string)   { d.add("scroll") }
func (d *testDelegate) EmitEvent(string, ...interface{}) {}

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
	for _, action := range []string{"screenshot", "send", "solve", "toggle", "clickthrough", "move_up", "move_down", "move_left", "move_right", "scroll_up", "scroll_down"} {
		s.handleTrigger(action)
	}
}
