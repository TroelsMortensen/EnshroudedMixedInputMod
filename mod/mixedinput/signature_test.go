package mixedinput_test

import (
	"testing"

	"github.com/trmo/EnshroudedMixInputMod/hook"
	"github.com/trmo/EnshroudedMixInputMod/mod/mixedinput"
	"github.com/trmo/EnshroudedMixInputMod/pattern"
	"github.com/trmo/EnshroudedMixInputMod/testing/sigtest"
)

func TestSignatures(t *testing.T) {
	sigtest.RunCases(t, []sigtest.Case{
		{
			Name:    "SUCCESS_UpdateInputModeFromEvent",
			Pattern: mixedinput.UpdateInputModeFromEventPattern,
			Err:     nil,
		},
		{
			Name:    "SUCCESS_UpdateGameplayAction",
			Pattern: mixedinput.UpdateGameplayActionPattern,
			Err:     nil,
		},
		{
			Name:    "SUCCESS_UpdateGameplayActionActiveStore",
			Pattern: mixedinput.UpdateGameplayActionActiveStorePattern,
			Err:     nil,
		},
		{
			Name:    "SUCCESS_AddDirectionalActionInput",
			Pattern: mixedinput.AddDirectionalActionInputPattern,
			Err:     nil,
		},
		{
			Name:    "SUCCESS_GetGameplayActionBinding",
			Pattern: mixedinput.GetGameplayActionBindingPattern,
			Err:     nil,
		},
		{
			Name:    "SUCCESS_UpdateGameplayActionGamepadHalf",
			Pattern: mixedinput.UpdateGameplayActionGamepadHalfPattern,
			Err:     nil,
		},
		{
			Name:    "FAIL_AmbiguousPattern",
			Pattern: pattern.MustParse("48 8B 07"),
			Err:     hook.ErrAmbiguousMatch,
		},
		{
			Name:    "FAIL_UnknownFunctionPattern",
			Pattern: pattern.MustParse("de ad be ef de ad be ef de ad be ef de ad be ef"),
			Err:     hook.ErrPatternNotFound,
		},
	})
}
