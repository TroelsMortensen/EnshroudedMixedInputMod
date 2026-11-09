package mixedinput

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"sync/atomic"
	"syscall"
	"unsafe"

	"github.com/trmo/EnshroudedMixInputMod/game"
	"github.com/trmo/EnshroudedMixInputMod/hook"
	"github.com/trmo/EnshroudedMixInputMod/keen"
	"github.com/trmo/EnshroudedMixInputMod/log"
)

const name = "mixed_input"

var (
	ErrJumpTableBaseDisagrees                 = errors.New("updateInputModeFromEvent jump table is not relative to the image base")
	ErrGamepadEventCaseDisagrees              = errors.New("updateInputModeFromEvent gamepad event case does not lead to the Gamepad store")
	ErrActiveStoreOutsideUpdateGameplayAction = errors.New("active state store is not in updateGameplayAction")
)

// MixedInput leaves the input mode unchanged on gamepad input, so it stays KeyboardMouse. In
// KeyboardMouse mode the gamepad bindings count alongside the keyboard mouse ones, and the gamepad
// stick adds its analog input to movement.
type MixedInput struct {
	log        *log.TraceLogger
	installErr error

	gamepadEventsKeepInputModePatch hook.CodePatch
	updateGameplayActionDetour      hook.Detour
	addDirectionalActionInputDetour hook.Detour

	evaluateGameplayActionHalf   uintptr
	actionEnabledOffset          uintptr
	actionActiveOffset           uintptr
	actionPreviouslyActiveOffset uintptr
	bindingLayout                BindingLayout

	gamepadModeSeen   atomic.Bool
	stickMovementSeen atomic.Bool
}

func New() *MixedInput {
	return &MixedInput{
		log: log.TraceLoggerWith("mod", name),
	}
}

func (*MixedInput) Name() string {
	return name
}

func (m *MixedInput) Install() error {
	m.installErr = m.install()

	return m.installErr
}

func (m *MixedInput) Installed() (bool, error) {
	return m.gamepadEventsKeepInputModePatch.Installed() &&
		m.updateGameplayActionDetour.Installed() &&
		m.addDirectionalActionInputDetour.Installed(), m.installErr
}

func (m *MixedInput) Remove() error {
	return errors.Join(
		m.addDirectionalActionInputDetour.Restore(),
		m.updateGameplayActionDetour.Restore(),
		m.gamepadEventsKeepInputModePatch.Restore(),
	)
}

func (m *MixedInput) install() error {
	module, image, err := game.Module()
	if err != nil {
		return err
	}

	updateInputModeFromEvent, err := hook.FindUniquePattern(module, image, UpdateInputModeFromEventPattern)
	if err != nil {
		return fmt.Errorf("updateInputModeFromEvent: %w", err)
	}

	gamepadEventEntries, noChangeEntries, err := gamepadEventJumpTableEntries(module, image, updateInputModeFromEvent)
	if err != nil {
		return err
	}

	updateGameplayAction, err := hook.FindUniquePattern(module, image, UpdateGameplayActionPattern)
	if err != nil {
		return fmt.Errorf("updateGameplayAction: %w", err)
	}

	activeStore, err := hook.FindUniquePattern(module, image, UpdateGameplayActionActiveStorePattern)
	if err != nil {
		return fmt.Errorf("updateGameplayAction active state store: %w", err)
	}

	activeStoreFunction, err := hook.FunctionStart(module, image, activeStore)
	if err != nil {
		return fmt.Errorf("updateGameplayAction active state store: %w", err)
	}

	if activeStoreFunction != updateGameplayAction {
		return fmt.Errorf("%w: store at %#x, function at %#x", ErrActiveStoreOutsideUpdateGameplayAction, activeStore, updateGameplayAction)
	}

	addDirectionalActionInput, err := hook.FindUniquePattern(module, image, AddDirectionalActionInputPattern)
	if err != nil {
		return fmt.Errorf("addDirectionalActionInput: %w", err)
	}

	m.bindingLayout, err = readBindingLayout(module, image, addDirectionalActionInput, updateGameplayAction)
	if err != nil {
		return err
	}

	m.evaluateGameplayActionHalf = activeStore + evaluateCallEnd +
		uintptr(int32(binary.LittleEndian.Uint32(image[activeStore-module+evaluateCallRelOffset:])))
	m.actionEnabledOffset = uintptr(binary.LittleEndian.Uint32(image[updateGameplayAction-module+actionEnabledDispOffset:]))
	m.actionActiveOffset, m.actionPreviouslyActiveOffset = readActiveStoreOffsets(module, image, activeStore)

	err = m.gamepadEventsKeepInputModePatch.Apply(gamepadEventEntries, noChangeEntries)
	if err != nil {
		return fmt.Errorf("updateInputModeFromEvent jump table: %w", err)
	}

	err = m.updateGameplayActionDetour.Apply(updateGameplayAction, syscall.NewCallback(m.updateGameplayAction))
	if err != nil {
		return errors.Join(fmt.Errorf("updateGameplayAction: %w", err), m.gamepadEventsKeepInputModePatch.Restore())
	}

	err = m.addDirectionalActionInputDetour.Apply(addDirectionalActionInput, syscall.NewCallback(m.addDirectionalActionInput))
	if err != nil {
		return errors.Join(fmt.Errorf("addDirectionalActionInput: %w", err),
			m.updateGameplayActionDetour.Restore(), m.gamepadEventsKeepInputModePatch.Restore())
	}

	m.log.Debug("hooked input mode and gameplay action evaluation",
		"updateInputModeFromEvent", fmt.Sprintf("%#x", updateInputModeFromEvent),
		"gamepadEventEntries", fmt.Sprintf("%#x", gamepadEventEntries),
		"updateGameplayAction", fmt.Sprintf("%#x", updateGameplayAction),
		"evaluateGameplayActionHalf", fmt.Sprintf("%#x", m.evaluateGameplayActionHalf),
		"actionEnabledOffset", fmt.Sprintf("%#x", m.actionEnabledOffset),
		"actionActiveOffset", fmt.Sprintf("%#x", m.actionActiveOffset),
		"actionPreviouslyActiveOffset", fmt.Sprintf("%#x", m.actionPreviouslyActiveOffset),
		"gamepadHalf", m.bindingLayout.GamepadHalf,
		"gamepadBindings", fmt.Sprintf("%d/%d", m.bindingLayout.GamepadPrimaryBinding, m.bindingLayout.GamepadSecondaryBinding),
		"addDirectionalActionInput", fmt.Sprintf("%#x", addDirectionalActionInput))

	return nil
}

// readActiveStoreOffsets returns the active and previously active offsets within an action, both
// stored by updateGameplayAction at activeStore.
func readActiveStoreOffsets(module uintptr, image []byte, activeStore uintptr) (uintptr, uintptr) {
	store := activeStore - module

	return uintptr(binary.LittleEndian.Uint32(image[store+activeStoreDispOffset:])),
		uintptr(binary.LittleEndian.Uint32(image[store+previouslyActiveStoreDispOffset:]))
}

// inputModeJumpTable is updateInputModeFromEvent's switch over event types 1..16. All values are RVAs.
type inputModeJumpTable struct {
	table             uintptr
	noChangeCase      uintptr
	keyboardMouseCase uintptr
	entries           [jumpTableEventTypeCount]uintptr
}

func readInputModeJumpTable(module uintptr, image []byte, updateInputModeFromEvent uintptr) (inputModeJumpTable, error) {
	function := updateInputModeFromEvent - module

	imageBase := updateInputModeFromEvent + imageBaseLeaEnd +
		uintptr(int32(binary.LittleEndian.Uint32(image[function+imageBaseLeaDispOffset:])))
	if imageBase != module {
		return inputModeJumpTable{}, fmt.Errorf("%w: %#x, module at %#x", ErrJumpTableBaseDisagrees, imageBase, module)
	}

	t := inputModeJumpTable{
		table:             uintptr(binary.LittleEndian.Uint32(image[function+jumpTableDispOffset:])),
		noChangeCase:      function + noChangeJumpEnd + uintptr(int8(image[function+noChangeJumpRelOffset])),
		keyboardMouseCase: function + keyboardMouseCaseOffset,
	}

	for i := range t.entries {
		t.entries[i] = uintptr(binary.LittleEndian.Uint32(image[t.entryOffset(jumpTableFirstEventType+keen.InputEventType(i)):]))
	}

	return t, nil
}

func (t inputModeJumpTable) entryOffset(eventType keen.InputEventType) uintptr {
	return t.table + uintptr(eventType-jumpTableFirstEventType)*jumpTableEntrySize
}

func (t inputModeJumpTable) entry(eventType keen.InputEventType) uintptr {
	return t.entries[eventType-jumpTableFirstEventType]
}

// gamepadEventJumpTableEntries returns the address of the ButtonDown..AnalogStick jump table entries
// and their replacement, all pointing at the no change case.
func gamepadEventJumpTableEntries(module uintptr, image []byte, updateInputModeFromEvent uintptr) (uintptr, []byte, error) {
	t, err := readInputModeJumpTable(module, image, updateInputModeFromEvent)
	if err != nil {
		return 0, nil, err
	}

	gamepadCase := t.entry(keen.InputEventTypeButtonDown)

	_, stored := scanCase(image, gamepadCase)
	if len(stored) != 1 || stored[0] != keen.InputModeGamepad {
		return 0, nil, fmt.Errorf("%w: ButtonDown case at %#x stores %v",
			ErrGamepadEventCaseDisagrees, module+gamepadCase, stored)
	}

	count := int(keen.InputEventTypeAnalogStick-keen.InputEventTypeButtonDown) + 1

	for i := range count {
		eventType := keen.InputEventTypeButtonDown + keen.InputEventType(i)
		if t.entry(eventType) != gamepadCase {
			return 0, nil, fmt.Errorf("%w: event type %d leads to %#x, Gamepad store at %#x",
				ErrGamepadEventCaseDisagrees, eventType, module+t.entry(eventType), module+gamepadCase)
		}
	}

	return module + t.entryOffset(keen.InputEventTypeButtonDown),
		bytes.Repeat(binary.LittleEndian.AppendUint32(nil, uint32(t.noChangeCase)), count), nil
}

// In KeyboardMouse mode the gamepad half also counts towards the action's active state. The
// original has already updated the gamepad bindings, it only skips evaluating their half.
func (m *MixedInput) updateGameplayAction(action, mode, keyboardMouseModifiers, gamepadModifiers, index, time uintptr) uintptr {
	m.updateGameplayActionDetour.CallOriginal(action, mode, keyboardMouseModifiers, gamepadModifiers, index, time)

	if keen.InputMode(mode) == keen.InputModeGamepad {
		if m.gamepadModeSeen.CompareAndSwap(false, true) {
			m.log.Warn("input mode is Gamepad although gamepad events are patched to keep KeyboardMouse")
		}

		return 0
	}

	if keen.InputMode(mode) != keen.InputModeKeyboardMouse ||
		*(*byte)(unsafe.Pointer(action + m.actionEnabledOffset)) == 0 {
		return 0
	}

	gamepadActive, _, _ := syscall.SyscallN(m.evaluateGameplayActionHalf, action+m.bindingLayout.GamepadHalf, index, time)
	if byte(gamepadActive) == 0 {
		return 0
	}

	active := (*byte)(unsafe.Pointer(action + m.actionActiveOffset))
	if *active == 0 && *(*byte)(unsafe.Pointer(action + m.actionPreviouslyActiveOffset)) == 0 {
		m.log.Trace("gamepad half activated action", "index", byte(index), "action", fmt.Sprintf("%#x", action))
	}

	*active = 1

	return 0
}

// In KeyboardMouse mode a movement action's gamepad stick or axis binding adds its analog input
// after the keyboard mouse one. Gamepad buttons are left out so they do not add to the keys.
func (m *MixedInput) addDirectionalActionInput(input, actions, action, primary, dt uintptr) uintptr {
	result := m.addDirectionalActionInputDetour.CallOriginal(input, actions, action, primary, dt)

	gameplayAction := keen.GameplayAction(action)
	if *(*keen.InputMode)(unsafe.Pointer(actions)) != keen.InputModeKeyboardMouse ||
		gameplayAction > keen.GameplayActionMovementDown {
		return result
	}

	actionStruct := actions + keen.ActionsHeaderSize + keen.ActionSize*uintptr(gameplayAction)
	if *(*byte)(unsafe.Pointer(actionStruct + m.actionActiveOffset)) == 0 {
		return result
	}

	binding := actionStruct + m.bindingLayout.GamepadSecondaryBinding
	if byte(primary) != 0 {
		binding = actionStruct + m.bindingLayout.GamepadPrimaryBinding
	}

	kind := *(*keen.BindingSourceKind)(unsafe.Pointer(binding + keen.BindingSourceKindOffset))
	if kind != keen.BindingSourceGamepadStick && kind != keen.BindingSourceGamepadAxis {
		return result
	}

	value := *(*float32)(unsafe.Pointer(binding + keen.BindingSnapshotValueOffset))

	addGamepadAnalogMovement((*DirectionalInput)(unsafe.Pointer(input)), gameplayAction, value)

	if value != 0 && m.stickMovementSeen.CompareAndSwap(false, true) {
		m.log.Debug("gamepad stick added to movement input", "action", byte(gameplayAction), "kind", byte(kind), "value", value)
	}

	return result
}
