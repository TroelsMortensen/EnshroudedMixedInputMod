package mixedinput_test

import (
	"testing"

	"github.com/trmo/EnshroudedMixInputMod/game"
	"github.com/trmo/EnshroudedMixInputMod/hook"
	"github.com/trmo/EnshroudedMixInputMod/keen"
	"github.com/trmo/EnshroudedMixInputMod/mod/mixedinput"
	"github.com/trmo/EnshroudedMixInputMod/testing/sigtest"
)

// The offsets read from getGameplayActionBinding and updateGameplayAction match the keen constants.
func TestBindingLayout(t *testing.T) {
	module, image := sigtest.LoadModule(t, game.ModuleName)

	addDirectionalActionInput, err := hook.FindUniquePattern(module, image, mixedinput.AddDirectionalActionInputPattern)
	if err != nil {
		t.Fatalf("addDirectionalActionInput: %v", err)
	}

	updateGameplayAction, err := hook.FindUniquePattern(module, image, mixedinput.UpdateGameplayActionPattern)
	if err != nil {
		t.Fatalf("updateGameplayAction: %v", err)
	}

	layout, err := mixedinput.ReadBindingLayout(module, image, addDirectionalActionInput, updateGameplayAction)
	if err != nil {
		t.Fatalf("ReadBindingLayout: %v", err)
	}

	t.Logf("%+v", layout)

	expected := mixedinput.BindingLayout{
		GamepadHalf:             keen.ActionGamepadHalf,
		GamepadPrimaryBinding:   keen.ActionGamepadPrimaryBinding,
		GamepadSecondaryBinding: keen.ActionGamepadSecondaryBinding,
		KeyboardMousePrimary:    keen.ActionKeyboardMousePrimaryBinding,
		KeyboardMouseSecondary:  keen.ActionKeyboardMouseSecondaryBinding,
	}

	if layout != expected {
		t.Fatalf("got %+v, expected %+v", layout, expected)
	}

	if keen.ActionGamepadHalf != keen.ActionHalfSize ||
		keen.ActionKeyboardMouseSecondaryBinding-keen.ActionKeyboardMousePrimaryBinding != keen.BindingSize {
		t.Fatalf("half size %d or binding size %d disagree with the offsets", keen.ActionHalfSize, keen.BindingSize)
	}
}

// The active state offsets stored by updateGameplayAction match the keen constants.
func TestActiveStoreOffsets(t *testing.T) {
	module, image := sigtest.LoadModule(t, game.ModuleName)

	activeStore, err := hook.FindUniquePattern(module, image, mixedinput.UpdateGameplayActionActiveStorePattern)
	if err != nil {
		t.Fatalf("updateGameplayAction active state store: %v", err)
	}

	active, previouslyActive := mixedinput.ReadActiveStoreOffsets(module, image, activeStore)
	if active != keen.ActionActive || previouslyActive != keen.ActionPreviouslyActive {
		t.Fatalf("got active %#x, previously active %#x, expected %#x, %#x",
			active, previouslyActive, keen.ActionActive, keen.ActionPreviouslyActive)
	}
}
