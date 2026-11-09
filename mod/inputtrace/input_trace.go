// Package inputtrace logs which input events reach updateInputModeFromEvent and which of them change
// the input mode. Diagnostic only, see notes/input-mode-events.md.
package inputtrace

import (
	"fmt"
	"sync"
	"syscall"
	"unsafe"

	"github.com/trmo/EnshroudedMixInputMod/game"
	"github.com/trmo/EnshroudedMixInputMod/hook"
	"github.com/trmo/EnshroudedMixInputMod/keen"
	"github.com/trmo/EnshroudedMixInputMod/log"
	"github.com/trmo/EnshroudedMixInputMod/mod/mixedinput"
)

const name = "input_trace"

const (
	eventTypeOffset = 0x0c
	eventDumpSize   = 0x30
	eventFirstFloat = 0x1c
	eventFloatCount = 5
)

type InputTrace struct {
	log        *log.TraceLogger
	installErr error

	updateInputModeFromEventDetour hook.Detour

	mtx   sync.Mutex
	seen  [256]bool
	flips int
}

func New() *InputTrace {
	return &InputTrace{
		log: log.TraceLoggerWith("mod", name),
	}
}

func (*InputTrace) Name() string {
	return name
}

func (t *InputTrace) Install() error {
	t.installErr = t.install()

	return t.installErr
}

func (t *InputTrace) Installed() (bool, error) {
	return t.updateInputModeFromEventDetour.Installed(), t.installErr
}

func (t *InputTrace) Remove() error {
	return t.updateInputModeFromEventDetour.Restore()
}

func (t *InputTrace) install() error {
	module, image, err := game.Module()
	if err != nil {
		return err
	}

	updateInputModeFromEvent, err := hook.FindUniquePattern(module, image, mixedinput.UpdateInputModeFromEventPattern)
	if err != nil {
		return fmt.Errorf("updateInputModeFromEvent: %w", err)
	}

	err = t.updateInputModeFromEventDetour.Apply(updateInputModeFromEvent, syscall.NewCallback(t.updateInputModeFromEvent))
	if err != nil {
		return fmt.Errorf("updateInputModeFromEvent: %w", err)
	}

	t.log.Info("tracing input events", "updateInputModeFromEvent", fmt.Sprintf("%#x", updateInputModeFromEvent))

	return nil
}

func (t *InputTrace) updateInputModeFromEvent(actions, event uintptr) uintptr {
	before := *(*keen.InputMode)(unsafe.Pointer(actions))

	t.updateInputModeFromEventDetour.CallOriginal(actions, event)

	after := *(*keen.InputMode)(unsafe.Pointer(actions))
	eventType := *(*keen.InputEventType)(unsafe.Pointer(event + eventTypeOffset))

	t.mtx.Lock()
	first := !t.seen[eventType]
	t.seen[eventType] = true

	flip := before != after
	if flip {
		t.flips++
	}

	flips := t.flips
	t.mtx.Unlock()

	if first {
		t.log.Info("first input event of type",
			"type", eventType, "id", byte(eventType), "mode", after,
			"event", fmt.Sprintf("% x", unsafe.Slice((*byte)(unsafe.Pointer(event)), eventDumpSize)))
	}

	if first && eventType == keen.InputEventTypeAnalogStick {
		t.logBindingKinds(actions)
	}

	if flip {
		t.log.Info("input mode changed",
			"type", eventType, "id", byte(eventType), "from", before, "to", after, "count", flips,
			"floats", fmt.Sprint(unsafe.Slice((*float32)(unsafe.Pointer(event+eventFirstFloat)), eventFloatCount)))
	}

	return 0
}

// logBindingKinds shows the binding source kinds of the movement and camera actions, once bindings are loaded.
func (t *InputTrace) logBindingKinds(actions uintptr) {
	kind := func(action keen.GameplayAction, binding uintptr) keen.BindingSourceKind {
		actionStruct := actions + keen.ActionsHeaderSize + keen.ActionSize*uintptr(action)

		return *(*keen.BindingSourceKind)(unsafe.Pointer(actionStruct + binding + keen.BindingSourceKindOffset))
	}

	for action := keen.GameplayActionMovementLeft; action <= keen.GameplayActionCameraDown; action++ {
		t.log.Info("action binding kinds",
			"action", byte(action),
			"keyboardMouse", fmt.Sprintf("%d/%d",
				kind(action, keen.ActionKeyboardMousePrimaryBinding), kind(action, keen.ActionKeyboardMouseSecondaryBinding)),
			"gamepad", fmt.Sprintf("%d/%d",
				kind(action, keen.ActionGamepadPrimaryBinding), kind(action, keen.ActionGamepadSecondaryBinding)))
	}
}
