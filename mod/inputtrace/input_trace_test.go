package inputtrace_test

import (
	"testing"

	"github.com/trmo/EnshroudedMixInputMod/game"
	"github.com/trmo/EnshroudedMixInputMod/hook"
	"github.com/trmo/EnshroudedMixInputMod/mod/mixedinput"
	"github.com/trmo/EnshroudedMixInputMod/testing/sigtest"
)

const (
	prologueSize = 32
	eventTypeLen = 5 // movzx r9d, byte ptr [rdx+0Ch]
)

// The detour replaces only the event type load, which relocates unchanged.
func TestUpdateInputModeFromEventPrologueRelocates(t *testing.T) {
	module, image := sigtest.LoadModule(t, game.ModuleName)

	updateInputModeFromEvent, err := hook.FindUniquePattern(module, image, mixedinput.UpdateInputModeFromEventPattern)
	if err != nil {
		t.Fatalf("updateInputModeFromEvent: %v", err)
	}

	function := updateInputModeFromEvent - module
	prologue := image[function : function+prologueSize]

	reloc, err := hook.Relocate(prologue, updateInputModeFromEvent, updateInputModeFromEvent+0x10000)
	if err != nil {
		t.Fatalf("Relocate: %v", err)
	}

	if reloc.SrcLen != eventTypeLen {
		t.Fatalf("relocated %d bytes, expected %d", reloc.SrcLen, eventTypeLen)
	}

	if string(reloc.Code) != string(prologue[:eventTypeLen]) {
		t.Fatalf("relocated code % x, expected % x", reloc.Code, prologue[:eventTypeLen])
	}
}
