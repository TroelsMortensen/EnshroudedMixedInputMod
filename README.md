# EnshroudedMixInput

Mixed input for [Enshrouded](https://store.steampowered.com/app/1203220/Enshrouded/): keyboard + mouse + XInput analog stick, with the UI locked to keyboard/mouse.

**Status:** scaffolding derived from the upstream mixed-input mod. Behavior is still the upstream gamepad+mouse merge; KBM lock + stick merge comes next.

**Continuing work:** see [`srs.md`](srs.md) for goals, architecture notes, and incremental tasks with verify steps.

## Goals

- Allow keyboard input
- Allow XInput for the analog stick
- Allow mouse use
- Lock the UI to keyboard/mouse

## Input behavior

- The input mode stays keyboard/mouse; gamepad input never switches the UI or prompts.
- The XInput stick drives movement only, with analog speed, alongside WASD.
- The camera is controlled by the mouse. The stick does not feed the camera.
- Gamepad buttons trigger their bound actions (for example XInput A jumps).

## Build

See [SETUP.md](SETUP.md) for the Windows toolchain. From the repo root:

```powershell
just build          # Debug → dist/
just build Release
```

Outputs: `dist/EnshroudedMixInput.dll` and `dist/dinput8.dll`.

```powershell
just deploy         # copy into EMIM_GAME_DIR (default Steam Enshrouded path)
```

## Installation

1. Build (or obtain) the DLLs.
2. Copy `dinput8.dll` and `EnshroudedMixInput.dll` next to `enshrouded.exe`.
3. Do **not** install alongside the original EnshroudedMixedInput mod (same `dinput8.dll` slot).

Uninstall: delete those two DLLs from the game directory.

## Logs

Written to `EnshroudedMixInput.log` in the game directory.

- `EMIM_LOG_LEVEL`: `trace`, `debug`, `info` (default), `warn`, or `error`
- `EMIM_LOG_CONSOLE=1`: open a console window for the log
- `EMIM_SUPPRESS_LOADING`: skip loading the extension DLL
- `EMIM_GAME_DIR`: game install path for `just deploy` / signature tests

## License

[GPL-3.0](LICENSE.txt)

Derived from [EnshroudedMixedInput](https://github.com/Alia5/EnshroudedMixedInput) by Peter Repukat (Alia5). The original project remains under `inspiration/` as a read-only reference.
