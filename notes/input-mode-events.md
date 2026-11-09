# `updateInputModeFromEvent` event types

Source: static read of `enshrouded.exe` (35,476,040 bytes, Steam build installed 2026-11-09) via
`TestInputModeJumpTableMap` in [`mod/mixedinput/jump_table_test.go`](../mod/mixedinput/jump_table_test.go):

```powershell
$env:EMIM_GAME_DIR = "D:\SteamLibrary\steamapps\common\Enshrouded"
go test -count=1 -v -run InputModeJumpTableMap ./mod/mixedinput
```

## Function shape

```text
movzx r9d, byte [rdx+0Ch]       ; event type
lea   eax, [r9-1]
cmp   eax, 0Fh
ja    noChange                  ; type 0 and > 16
jmp   [jumpTable + (type-1)*4]  ; image-relative RVAs

keyboardMouse:  cmp byte [rcx], 0 / jz noChange / inc byte [rcx+0A490h] / mov byte [rcx], 0 / ret
gamepad:        cmp r9b, 5 / jnz gamepadStore
                ; AnalogStick only: skip unless max(|ev+1C - ev+24|, |ev+20 - ev+28|) >= ev+2C
gamepadStore:   cmp byte [rcx], 1 / jz noChange / inc byte [rcx+0A490h] / mov byte [rcx], 1 / ret
noChange:       ret
```

`rcx+0A490h` is the mode change counter. RVAs below are for this build: function `0xEC2B30`,
KeyboardMouse case `0xEC2B55`, Gamepad case `0xEC2B64`, no change `0xEC2BA8`.

`updateInputModeFromEvent` is a leaf with no unwind entry, so `hook.FunctionStart` cannot resolve it.

## Event type to case

Names come from the `keen` input event type name table (pointer/length pairs, `MouseDoubleClick` entry
at RVA `0x1FEF2F0`, so the table starts at RVA `0x1FEF210`). The address `0x141FCB210` in
[`keen/input.go`](../keen/input.go) is stale for this build.

| Type | Name | Target RVA | Case |
|-----:|------|-----------|------|
| 0 | ControllerConnected | (ja) | no change |
| 1 | ControllerDisconnected | `0xEC2BA8` | no change |
| 2 | ButtonDown | `0xEC2B64` | **Gamepad store** |
| 3 | ButtonUp | `0xEC2B64` | **Gamepad store** |
| 4 | Axis | `0xEC2B64` | **Gamepad store** |
| 5 | AnalogStick | `0xEC2B64` | **Gamepad store**, only past the movement threshold |
| 6 | CharacterInput | `0xEC2BA8` | no change |
| 7 | KeyDown | `0xEC2B55` | KeyboardMouse store |
| 8 | KeyUp | `0xEC2B55` | KeyboardMouse store |
| 9 | MouseDown | `0xEC2B55` | KeyboardMouse store |
| 10 | MouseUp | `0xEC2B55` | KeyboardMouse store |
| 11 | MouseMove | `0xEC2B55` | KeyboardMouse store |
| 12 | MouseRelativeMovement | `0xEC2B55` | KeyboardMouse store |
| 13 | MouseWheel | `0xEC2B55` | KeyboardMouse store |
| 14 | MouseDoubleClick | `0xEC2B55` | KeyboardMouse store |
| 15 | MouseLeft | `0xEC2BA8` | no change |
| 16 | FocusLost | `0xEC2BA8` | no change |
| 17+ | TouchBegin, TouchEnd, TouchCancel, TouchMove, SpaceNavigatorRelativeMovement, VirtualKeyboardOpened, ... | (ja) | no change |

## Confirmed in-game (Task 1.2, 2026-11-09)

Captured with the temporary `mod/inputtrace` detour (removed after Task 2.1): one session with an Azeron Keyzen, keyboard and mouse.

| Source | Event types seen | Mode change |
|--------|------------------|-------------|
| Keyzen stick | 5 AnalogStick only (no 4 Axis) | KeyboardMouse -> Gamepad, but only once the stick moves past the threshold |
| Keyzen button | 2 ButtonDown, 3 ButtonUp (button id 5) | Gamepad store (already in Gamepad during the test) |
| Keyboard | 7 KeyDown, 6 CharacterInput, 8 KeyUp | Gamepad -> KeyboardMouse on KeyDown |
| Mouse | 9 MouseDown, 10 MouseUp, 11 MouseMove, 12 MouseRelativeMovement | none; the upstream mouse patch was still active then (replaced by the gamepad patch in Task 2.1) |

The first stick event (a nudge) did not change the mode; a full push 1.3 s later did. Every flip came from
AnalogStick or KeyDown, so nothing else is toggling the mode.

### Event layout observed

| Offset | Meaning |
|--------|---------|
| `+0x00` | device class: `00` mouse, `01` keyboard, `03` gamepad |
| `+0x02` | device kind: `01` mouse, `02` keyboard, `04` gamepad |
| `+0x0A` | `01` for keyboard/mouse events, `00` for gamepad events |
| `+0x0C` | event type |
| `+0x10` | 64-bit value, rising across events (timestamp) |
| `+0x18` | button / key code (gamepad button `5`, key `0x37`; CharacterInput: character `0x65`) |
| `+0x1C`, `+0x20` | AnalogStick: first x, y pair (probably the current position, see [action-layout.md](action-layout.md)) |
| `+0x24`, `+0x28` | AnalogStick: second x, y pair (probably the previous position). The threshold compares it with the first pair |
| `+0x2C` | AnalogStick: mode-switch threshold `0.23953247`, which equals XInput's `XINPUT_GAMEPAD_LEFT_THUMB_DEADZONE` (7849 / 32767) |

Example flips: `[-0.0078, -0.0059] / [-0.7793, -0.3848]` and `[-0.0020, -0.0020] / [0.0547, 0.7969]`.

## Implications for Phase 2

- The only entries that switch the mode to Gamepad are types 2-5. Patching those four entries to the no-change `ret`
  is enough to stop the input mode from leaving KeyboardMouse.
- Patch all of 2-5, not just 5: the Keyzen also produces gamepad ButtonDown/ButtonUp.
- Mouse types 9-14 already store KeyboardMouse, so with our polarity the upstream mouse patch is not needed.
