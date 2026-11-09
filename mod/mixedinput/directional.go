package mixedinput

import "github.com/trmo/EnshroudedMixInputMod/keen"

// DirectionalInput holds an x, y pair per binding class: buttons at 0/1, gamepad analog at 2/3,
// mouse at 4/5 and special at 6/7.
type DirectionalInput [8]float32

const directionalSlotGamepadAnalog = 2

// addGamepadAnalogMovement mirrors addDirectionalActionInput for a movement action's stick or axis
// binding, whose axis value is the snapshot unscaled.
func addGamepadAnalogMovement(input *DirectionalInput, action keen.GameplayAction, value float32) {
	switch action {
	case keen.GameplayActionMovementLeft:
		input[directionalSlotGamepadAnalog] -= value
	case keen.GameplayActionMovementRight:
		input[directionalSlotGamepadAnalog] += value
	case keen.GameplayActionMovementUp:
		input[directionalSlotGamepadAnalog+1] += value
	case keen.GameplayActionMovementDown:
		input[directionalSlotGamepadAnalog+1] -= value
	}
}
