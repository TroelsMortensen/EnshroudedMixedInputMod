package mixedinput

import (
	"golang.org/x/arch/x86/x86asm"

	"github.com/trmo/EnshroudedMixInputMod/keen"
)

const caseScanLimit = 128

type InputModeEventCase byte

const (
	InputModeEventCaseOther InputModeEventCase = iota
	InputModeEventCaseNoChange
	InputModeEventCaseKeyboardMouse
	InputModeEventCaseGamepad
)

func (c InputModeEventCase) String() string {
	switch c {
	case InputModeEventCaseNoChange:
		return "no change"
	case InputModeEventCaseKeyboardMouse:
		return "KeyboardMouse store"
	case InputModeEventCaseGamepad:
		return "Gamepad store"
	default:
		return "other"
	}
}

// InputModeEvent is one updateInputModeFromEvent jump table entry. Target is an RVA, Code the
// target's instructions up to the first ret or jmp.
type InputModeEvent struct {
	Type   keen.InputEventType
	Target uintptr
	Case   InputModeEventCase
	Code   []byte
}

// inputModeEvents classifies every jump table entry by the keen::InputMode it stores into actions[0].
func inputModeEvents(module uintptr, image []byte, updateInputModeFromEvent uintptr) ([]InputModeEvent, error) {
	t, err := readInputModeJumpTable(module, image, updateInputModeFromEvent)
	if err != nil {
		return nil, err
	}

	events := make([]InputModeEvent, 0, len(t.entries))

	for i, target := range t.entries {
		event := InputModeEvent{
			Type:   jumpTableFirstEventType + keen.InputEventType(i),
			Target: target,
		}

		var stored []keen.InputMode
		event.Code, stored = scanCase(image, target)

		switch {
		case target == t.noChangeCase:
			event.Case = InputModeEventCaseNoChange
		case target == t.keyboardMouseCase:
			event.Case = InputModeEventCaseKeyboardMouse
		case len(stored) == 1 && stored[0] == keen.InputModeKeyboardMouse:
			event.Case = InputModeEventCaseKeyboardMouse
		case len(stored) == 1 && stored[0] == keen.InputModeGamepad:
			event.Case = InputModeEventCaseGamepad
		}

		events = append(events, event)
	}

	return events, nil
}

// scanCase decodes from offset up to the first ret or jmp and collects the immediates stored to byte [rcx].
func scanCase(image []byte, offset uintptr) ([]byte, []keen.InputMode) {
	stored := []keen.InputMode{}
	end := offset

	for end-offset < caseScanLimit {
		inst, err := x86asm.Decode(image[end:], 64)
		if err != nil {
			break
		}

		end += uintptr(inst.Len)

		if inst.Op == x86asm.MOV && isModeByte(inst.Args[0]) {
			imm, ok := inst.Args[1].(x86asm.Imm)
			if ok {
				stored = append(stored, keen.InputMode(imm))
			}
		}

		if inst.Op == x86asm.RET || inst.Op == x86asm.JMP {
			break
		}
	}

	return image[offset:end], stored
}

func isModeByte(arg x86asm.Arg) bool {
	mem, ok := arg.(x86asm.Mem)

	return ok && mem.Base == x86asm.RCX && mem.Index == 0 && mem.Disp == 0
}
