package keen

import "fmt"

// InputMode is keen::InputMode, reflected type at 0x141B69B80, values at 0x141B696B0.
type InputMode byte

const (
	InputModeKeyboardMouse InputMode = 0
	InputModeGamepad       InputMode = 1
)

// InputEventType is the input event type at event+0x0c, value names (pointer, length pairs) at RVA 0x1FEF210.
type InputEventType byte

const (
	InputEventTypeControllerConnected    InputEventType = 0
	InputEventTypeControllerDisconnected InputEventType = 1
	InputEventTypeButtonDown             InputEventType = 2
	InputEventTypeButtonUp               InputEventType = 3
	InputEventTypeAxis                   InputEventType = 4
	// InputEventTypeAnalogStick switches to Gamepad only once the stick moved past the event's threshold.
	InputEventTypeAnalogStick           InputEventType = 5
	InputEventTypeCharacterInput        InputEventType = 6
	InputEventTypeKeyDown               InputEventType = 7
	InputEventTypeKeyUp                 InputEventType = 8
	InputEventTypeMouseDown             InputEventType = 9
	InputEventTypeMouseUp               InputEventType = 10
	InputEventTypeMouseMove             InputEventType = 11
	InputEventTypeMouseRelativeMovement InputEventType = 12
	InputEventTypeMouseWheel            InputEventType = 13
	InputEventTypeMouseDoubleClick      InputEventType = 14
	InputEventTypeMouseLeft             InputEventType = 15
	InputEventTypeFocusLost             InputEventType = 16
)

var inputEventTypeNames = [...]string{
	"ControllerConnected", "ControllerDisconnected", "ButtonDown", "ButtonUp", "Axis", "AnalogStick",
	"CharacterInput", "KeyDown", "KeyUp", "MouseDown", "MouseUp", "MouseMove", "MouseRelativeMovement",
	"MouseWheel", "MouseDoubleClick", "MouseLeft", "FocusLost",
}

func (t InputEventType) String() string {
	if int(t) < len(inputEventTypeNames) {
		return inputEventTypeNames[t]
	}

	return fmt.Sprintf("InputEventType(%d)", byte(t))
}

func (m InputMode) String() string {
	switch m {
	case InputModeKeyboardMouse:
		return "KeyboardMouse"
	case InputModeGamepad:
		return "Gamepad"
	default:
		return fmt.Sprintf("InputMode(%d)", byte(m))
	}
}

const (
	ActionsHeaderSize = 8
	ActionSize        = 280

	// An action holds a keyboard mouse half at +0 and a gamepad half, each with a primary and a
	// secondary binding at +32 / +84 of the half.
	ActionHalfSize = 136
	BindingSize    = 52

	ActionKeyboardMousePrimaryBinding   = 32
	ActionKeyboardMouseSecondaryBinding = 84

	ActionGamepadHalf             = 136
	ActionGamepadPrimaryBinding   = 168
	ActionGamepadSecondaryBinding = 220

	ActionActive           = 272
	ActionPreviouslyActive = 273
)

const (
	BindingSourceKindOffset    = 0
	BindingSnapshotValueOffset = 32
)

type BindingSourceKind byte

const (
	BindingSourceUnbound               BindingSourceKind = 0
	BindingSourceKey                   BindingSourceKind = 1
	BindingSourceMouseButton           BindingSourceKind = 2
	BindingSourceMouseWheel            BindingSourceKind = 3
	BindingSourceMouseRelativeMovement BindingSourceKind = 4
	BindingSourceSpecial               BindingSourceKind = 5
	BindingSourceGamepadButton         BindingSourceKind = 6
	BindingSourceGamepadAxis           BindingSourceKind = 7
	BindingSourceGamepadStick          BindingSourceKind = 8
)

type GameplayAction byte

const (
	GameplayActionMovementLeft  GameplayAction = 0
	GameplayActionMovementRight GameplayAction = 1
	GameplayActionMovementUp    GameplayAction = 2
	GameplayActionMovementDown  GameplayAction = 3
	GameplayActionCameraLeft    GameplayAction = 4
	GameplayActionCameraRight   GameplayAction = 5
	GameplayActionCameraUp      GameplayAction = 6
	GameplayActionCameraDown    GameplayAction = 7

	GameplayActionCombatTargetSelect1dLeft  GameplayAction = 111
	GameplayActionCombatTargetSelect1dRight GameplayAction = 112
	GameplayActionCombatTargetSelect2dLeft  GameplayAction = 113
	GameplayActionCombatTargetSelect2dRight GameplayAction = 114
	GameplayActionCombatTargetSelect2dUp    GameplayAction = 115
	GameplayActionCombatTargetSelect2dDown  GameplayAction = 116
)
