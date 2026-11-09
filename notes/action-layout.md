# Gameplay action layout

Source: static read of `enshrouded.exe` (same build as [input-mode-events.md](input-mode-events.md)).
Asserted by `TestBindingLayout` in [`mod/mixedinput/binding_layout_test.go`](../mod/mixedinput/binding_layout_test.go).
The mod reads the same values at install (`readBindingLayout`) and refuses to install if they disagree.

## Actions table

```text
actions + 0x0000   InputMode (byte)
actions + 0x0008   action[0]            ; 141 actions (0x8D), 280 (0x118) bytes each
actions + 0x9A40   end of the action array
actions + 0xA490   input mode change counter
```

## One action (280 bytes, offsets from the action)

| Offset | Size | Field |
|-------:|-----:|-------|
| 0 | 136 | keyboard mouse half |
| 136 (0x88) | 136 | gamepad half |
| 272 (0x110) | 1 | active |
| 273 (0x111) | 1 | previously active |
| 274 (0x112) | 1 | enabled |

## One half (136 bytes, offsets from the half)

| Offset | Field |
|-------:|-------|
| 0x08 | pressed (byte) |
| 0x10 | press time (int64) |
| 0x18 | release time (int64) |
| 0x20 | primary binding (52 bytes) |
| 0x54 | secondary binding (52 bytes) |

So the keyboard mouse bindings are at action `+32` / `+84`, and the gamepad bindings are at action `+168` / `+220`.

## One binding (52 bytes, offsets from the binding)

| Offset | Field |
|-------:|-------|
| 0x00 | `BindingSourceKind` (byte) |
| 0x18 | flag tested before the modifier check |
| 0x19 | flag tested before the modifier check |
| 0x20 | snapshot value (float32) |
| 0x28 | modifier mask (uint32) |
| 0x2C | modifier value (uint32) |
| 0x30 | result: active |
| 0x31 | result: modifiers match |

## How it was found

`getGameplayActionBinding(actions, action, primary)`, called from `addDirectionalActionInput`:

```text
movzx eax, dl
movzx edx, byte ptr [rcx]          ; InputMode
imul  r9, rax, 118h
add   r9, rcx                      ; actions + index * 280 (header still included)
test  edx, edx / jz keyboardMouse
cmp   edx, 1 / jz gamepad
lea   rax, [r9+0E4h] / ret         ; any other mode: gamepad secondary
gamepad:        primary ? 0B0h : 0E4h   -> action +168 / +220
keyboardMouse:  primary ? 28h  : 5Ch    -> action +32  / +84
```

`updateGameplayAction(action, ...)`, keyboard mouse half in `rsi`, gamepad half in `r12`:

```text
lea   rcx, [rsi+20h]   / call updateBinding    ; keyboard mouse primary
lea   rcx, [rsi+54h]   / call updateBinding    ; keyboard mouse secondary
lea   r12, [rsi+88h]                           ; gamepad half
lea   rcx, [r12+20h]   / call updateBinding    ; gamepad primary   (+168)
lea   r15, [rsi+0DCh]  / call updateBinding    ; gamepad secondary (+220)
...
mode == KeyboardMouse ? rsi : r12 -> evaluateGameplayActionHalf(half, index, time) -> [rsi+110h] active
```

Confidence: both functions agree independently on all four binding offsets. The binding field
meanings at `0x18..0x31` are inferred from `updateGameplayAction` and are not needed yet.

## Confirmed in-game (2026-11-09)

The temporary `mod/inputtrace` (removed after Task 2.1) dump of the live actions table on the first AnalogStick event, in-world, default bindings:

| Actions | Keyboard mouse primary / secondary | Gamepad primary / secondary |
|---------|------------------------------------|-----------------------------|
| 0-3 movement | 1 Key / 1 Key | 8 GamepadStick / 0 Unbound |
| 4-7 camera | 4 MouseRelativeMovement / 0 Unbound | 8 GamepadStick / 0 Unbound |

Every value is a valid `BindingSourceKind` of the expected class, so the gamepad offsets `+168` / `+220` are confirmed.
The movement and camera gamepad bindings are sticks (8), not axes (7), so Phase 3 merges `BindingSourceGamepadStick` first.

AnalogStick event pairs (for Phase 3): pushing the stick up gave `[-0.04, 0.69]` / `[-0.04, 0.30]` at `+0x1C` / `+0x24`.
That suggests `+0x1C`/`+0x20` is the current position and `+0x24`/`+0x28` the previous one, but this is not proven.
