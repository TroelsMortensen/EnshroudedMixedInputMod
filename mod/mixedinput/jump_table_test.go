package mixedinput_test

import (
	"bytes"
	"encoding/binary"
	"testing"

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
