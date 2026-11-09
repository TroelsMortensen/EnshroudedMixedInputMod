package mixedinput_test

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
	"testing"

	"golang.org/x/arch/x86/x86asm"

	"github.com/trmo/EnshroudedMixInputMod/game"
	"github.com/trmo/EnshroudedMixInputMod/hook"
	"github.com/trmo/EnshroudedMixInputMod/keen"
	"github.com/trmo/EnshroudedMixInputMod/mod/mixedinput"
	"github.com/trmo/EnshroudedMixInputMod/testing/sigtest"
)

const (
	jumpTableEntrySize = 4
	retOpcode          = 0xC3
)

// The mouse event entries are rewritten to the no change case, which is updateInputModeFromEvent's ret.
func TestMouseEventJumpTableEntries(t *testing.T) {
	module, image := sigtest.LoadModule(t, game.ModuleName)

	updateInputModeFromEvent, err := hook.FindUniquePattern(module, image, mixedinput.UpdateInputModeFromEventPattern)
	if err != nil {
		t.Fatalf("updateInputModeFromEvent: %v", err)
	}

	_, replacement, err := mixedinput.MouseEventJumpTableEntries(module, image, updateInputModeFromEvent)
	if err != nil {
		t.Fatalf("MouseEventJumpTableEntries: %v", err)
	}

	count := int(keen.InputEventTypeMouseDoubleClick-keen.InputEventTypeMouseDown) + 1
	expected := bytes.Repeat(replacement[:jumpTableEntrySize], count)

	if !bytes.Equal(replacement, expected) {
		t.Fatalf("got % x, expected % x", replacement, expected)
	}

	noChangeCase := binary.LittleEndian.Uint32(replacement)
	if image[noChangeCase] != retOpcode {
		t.Fatalf("no change case at %#x is %#x, expected ret", module+uintptr(noChangeCase), image[noChangeCase])
	}
}

// Logs every updateInputModeFromEvent case for notes/input-mode-events.md.
func TestInputModeJumpTableMap(t *testing.T) {
	module, image := sigtest.LoadModule(t, game.ModuleName)

	updateInputModeFromEvent, err := hook.FindUniquePattern(module, image, mixedinput.UpdateInputModeFromEventPattern)
	if err != nil {
		t.Fatalf("updateInputModeFromEvent: %v", err)
	}

	events, err := mixedinput.InputModeEvents(module, image, updateInputModeFromEvent)
	if err != nil {
		t.Fatalf("InputModeEvents: %v", err)
	}

	_, replacement, err := mixedinput.MouseEventJumpTableEntries(module, image, updateInputModeFromEvent)
	if err != nil {
		t.Fatalf("MouseEventJumpTableEntries: %v", err)
	}

	// updateInputModeFromEvent is a leaf without unwind info; its cases lie between its start and the shared ret.
	function := updateInputModeFromEvent - module
	ret := uintptr(binary.LittleEndian.Uint32(replacement))

	for _, event := range events {
		t.Logf("event type %2d -> %#x (rva %#x) %-19s %s",
			event.Type, module+event.Target, event.Target, event.Case, disassemble(module, event))

		if event.Target < function || event.Target > ret {
			t.Errorf("event type %d leads to %#x, outside updateInputModeFromEvent %#x..%#x",
				event.Type, module+event.Target, module+function, module+ret)
		}

		mouse := event.Type >= keen.InputEventTypeMouseDown && event.Type <= keen.InputEventTypeMouseDoubleClick
		if mouse && event.Case != mixedinput.InputModeEventCaseKeyboardMouse {
			t.Errorf("mouse event type %d is %s, expected %s",
				event.Type, event.Case, mixedinput.InputModeEventCaseKeyboardMouse)
		}
	}
}

func disassemble(module uintptr, event mixedinput.InputModeEvent) string {
	parts := []string{}

	for offset := 0; offset < len(event.Code); {
		inst, err := x86asm.Decode(event.Code[offset:], 64)
		if err != nil {
			parts = append(parts, fmt.Sprintf("% x", event.Code[offset:]))

			break
		}

		pc := uint64(module + event.Target + uintptr(offset))
		parts = append(parts, x86asm.IntelSyntax(inst, pc, nil))
		offset += inst.Len
	}

	return strings.Join(parts, "; ")
}
