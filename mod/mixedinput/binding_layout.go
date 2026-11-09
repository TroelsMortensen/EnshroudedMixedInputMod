package mixedinput

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/trmo/EnshroudedMixInputMod/hook"
	"github.com/trmo/EnshroudedMixInputMod/keen"
)

var (
	ErrBindingCallDisagrees   = errors.New("addDirectionalActionInput does not call getGameplayActionBinding")
	ErrGamepadHalfOutside     = errors.New("gamepad half setup is not in updateGameplayAction")
	ErrBindingLayoutDisagrees = errors.New("action binding layout disagrees")
)

// BindingLayout holds offsets within an action (actions + header + index * action size).
type BindingLayout struct {
	GamepadHalf             uintptr
	GamepadPrimaryBinding   uintptr
	GamepadSecondaryBinding uintptr
	KeyboardMousePrimary    uintptr
	KeyboardMouseSecondary  uintptr
}

func readBindingLayout(module uintptr, image []byte, addDirectionalActionInput, updateGameplayAction uintptr) (BindingLayout, error) {
	getBinding, err := hook.FindUniquePattern(module, image, GetGameplayActionBindingPattern)
	if err != nil {
		return BindingLayout{}, fmt.Errorf("getGameplayActionBinding: %w", err)
	}

	called := addDirectionalActionInput + getBindingCallEnd +
		uintptr(int32(binary.LittleEndian.Uint32(image[addDirectionalActionInput-module+getBindingCallRelOffset:])))
	if called != getBinding {
		return BindingLayout{}, fmt.Errorf("%w: calls %#x, getGameplayActionBinding at %#x", ErrBindingCallDisagrees, called, getBinding)
	}

	gamepadHalfSetup, err := hook.FindUniquePattern(module, image, UpdateGameplayActionGamepadHalfPattern)
	if err != nil {
		return BindingLayout{}, fmt.Errorf("updateGameplayAction gamepad half: %w", err)
	}

	setupFunction, err := hook.FunctionStart(module, image, gamepadHalfSetup)
	if err != nil {
		return BindingLayout{}, fmt.Errorf("updateGameplayAction gamepad half: %w", err)
	}

	if setupFunction != updateGameplayAction {
		return BindingLayout{}, fmt.Errorf("%w: setup at %#x, function at %#x", ErrGamepadHalfOutside, gamepadHalfSetup, updateGameplayAction)
	}

	imm := func(at, offset uintptr) uintptr {
		return uintptr(binary.LittleEndian.Uint32(image[at-module+offset:]))
	}

	// getGameplayActionBinding offsets are from actions + index * action size, so they include the header.
	fromAction := func(offset uintptr) uintptr {
		return imm(getBinding, offset) - keen.ActionsHeaderSize
	}

	layout := BindingLayout{
		GamepadHalf:             imm(gamepadHalfSetup, gamepadHalfDispOffset),
		GamepadPrimaryBinding:   fromAction(getBindingGamepadPrimaryOffset),
		GamepadSecondaryBinding: fromAction(getBindingGamepadSecondaryOffset),
		KeyboardMousePrimary:    fromAction(getBindingKeyboardMousePrimaryOffset),
		KeyboardMouseSecondary:  fromAction(getBindingKeyboardMouseSecondaryOffset),
	}

	stride := imm(getBinding, getBindingStrideOffset)

	switch {
	case stride != keen.ActionSize:
		return layout, fmt.Errorf("%w: action size %d, expected %d", ErrBindingLayoutDisagrees, stride, keen.ActionSize)
	case layout.KeyboardMousePrimary != keen.ActionKeyboardMousePrimaryBinding ||
		layout.KeyboardMouseSecondary != keen.ActionKeyboardMouseSecondaryBinding:
		return layout, fmt.Errorf("%w: keyboard mouse bindings at %d / %d, expected %d / %d", ErrBindingLayoutDisagrees,
			layout.KeyboardMousePrimary, layout.KeyboardMouseSecondary,
			keen.ActionKeyboardMousePrimaryBinding, keen.ActionKeyboardMouseSecondaryBinding)
	case layout.GamepadPrimaryBinding-layout.GamepadHalf != keen.ActionKeyboardMousePrimaryBinding ||
		layout.GamepadSecondaryBinding-layout.GamepadHalf != keen.ActionKeyboardMouseSecondaryBinding:
		return layout, fmt.Errorf("%w: gamepad bindings at %d / %d are not at +%d / +%d of the half at %d", ErrBindingLayoutDisagrees,
			layout.GamepadPrimaryBinding, layout.GamepadSecondaryBinding,
			keen.ActionKeyboardMousePrimaryBinding, keen.ActionKeyboardMouseSecondaryBinding, layout.GamepadHalf)
	}

	return layout, nil
}
