# EnshroudedMixInput — Software Requirements Specification

**Document purpose:** handoff for continuing work in a fresh chat.  
**Status:** Project done (all phases 0–4 complete, definition of done in section 10 met). Gamepad events no longer change the input mode, so the game stays in KeyboardMouse. Gamepad buttons trigger their actions (XInput A jumps), the stick gives analog movement alongside WASD, and the mouse keeps the camera. Uninstalling restores vanilla behavior.  
**Last updated:** 2026-11-09

---

## 1. Problem

[Enshrouded](https://store.steampowered.com/app/1203220/Enshrouded/) supports **either** controller **or** mouse/keyboard, not both at once. Touching a controller switches UI + bindings to gamepad; touching mouse/keyboard switches back to KBM. The switch stutters and refreshes UI.

**Hardware goal:** Azeron Keyzen (XInput analog stick) + normal keyboard keys + mouse.

**What fails without a mod:** stick activity forces gamepad mode → keyboard stops feeling native / UI flips; without stick, you lose analog movement.

**Upstream inspiration:** [Alia5/EnshroudedMixedInput](https://github.com/Alia5/EnshroudedMixedInput) (GPL-3.0), vendored read-only at [`inspiration/`](inspiration/). It keeps **gamepad mode** and lets **mouse** work without switching to KBM. Keyboard still switches mode. That is the opposite polarity of what we need.

---

## 2. Goals (acceptance criteria)

When the mod is installed next to `enshrouded.exe`:

| ID | Requirement | Done when |
|----|-------------|-----------|
| G1 | Keyboard input works | Keys drive actions while stick is also in use; no forced gamepad UI |
| G2 | Mouse input works | Look / buttons / wheel work without mode thrashing |
| G3 | XInput analog stick works | Stick contributes analog movement (and later camera if desired) while staying in KBM mode |
| G4 | UI locked to keyboard/mouse | Prompts and input mode stay KBM; touching stick does **not** flip to gamepad UI |

**Non-goals (for now):**

- Full gamepad-button parity while locked to KBM UI (stick/axes first)
- Supporting two mods at once (same `dinput8.dll` slot)
- Linux/Proton packaging polish (Windows primary)
- Changing `inspiration/` (read-only reference)

---

## 3. Current project state

### 3.1 Layout

```text
EnshroudedMixInputMod/          ← active Go project (repo root)
  inspiration/                  ← READ-ONLY copy of upstream
  emi/extension/                ← c-shared DLL entry
  loader/                       ← dinput8.dll proxy
  hook/ pattern/ game/ keen/    ← RE + patching infrastructure
  mod/mixedinput/               ← input behavior (still upstream logic)
  justfile SETUP.md README.md
  srs.md                        ← this document
```

### 3.2 Identity

| Item | Value |
|------|--------|
| Go module | `github.com/trmo/EnshroudedMixInputMod` |
| Extension DLL | `EnshroudedMixInput.dll` |
| Loader DLL | `dinput8.dll` |
| Log file | `EnshroudedMixInput.log` |
| Env prefix | `EMIM_` (`EMIM_LOG_LEVEL`, `EMIM_LOG_CONSOLE`, `EMIM_SUPPRESS_LOADING`, `EMIM_GAME_DIR`) |
| License | GPL-3.0; derived from EnshroudedMixedInput |

### 3.3 Build / deploy

```powershell
just build          # → dist/EnshroudedMixInput.dll + dist/dinput8.dll
just build Release
just deploy         # copies into EMIM_GAME_DIR (default Steam Enshrouded path)
```

Toolchain: see [`SETUP.md`](SETUP.md). `justfile` uses `-G "MinGW Makefiles"` for LLVM-MinGW.

### 3.4 What the code does today

In [`mod/mixedinput/mixed_input.go`](mod/mixedinput/mixed_input.go):

1. **Jump-table patch** on `updateInputModeFromEvent`: gamepad event types **2–5** → “no change” (do not switch to Gamepad). Mouse types 9–14 are unpatched and store KBM as in vanilla. (Task 2.1)
2. **`updateGameplayAction` detour (inverted, Task 3.1):** if mode == **KeyboardMouse**, also evaluate the **gamepad** half and set active. If it ever sees Gamepad mode it only logs a one-time warning (Task 2.2 guard).
3. **`addDirectionalActionInput` detour (inverted, Task 3.2):** if mode == **KeyboardMouse**, movement actions 0–3 also add their **gamepad stick/axis** value into the gamepad analog slot.

Keen constants: [`keen/input.go`](keen/input.go) — `InputModeKeyboardMouse=0`, `InputModeGamepad=1`. KBM binding offsets `32` / `84` are known. Gamepad half `136` and gamepad binding offsets `168` / `220` were documented in Task 1.3 ([`notes/action-layout.md`](notes/action-layout.md)).

---

## 4. Target technical approach

**Invert the upstream design:**

| Concern | Upstream | Ours |
|---------|----------|------|
| Mode lock | Ignore mouse → stay Gamepad | Ignore gamepad events → stay **KeyboardMouse** |
| Merge | In Gamepad, inject KBM bindings | In **KeyboardMouse**, inject **gamepad** stick/axis bindings |
| UI | Gamepad prompts | **KBM prompts** |

Same hook points, flipped polarity:

```text
updateInputModeFromEvent  → patch gamepad event jump-table entries to "no change"
updateGameplayAction      → when mode==KBM, also evaluate gamepad half → set active
addDirectionalActionInput → when mode==KBM, add gamepad stick/axis directional values
```

### 4.1 Known RE facts (from patterns / keen)

- `updateInputModeFromEvent(actions, event*)`: `actions[0]` = `InputMode`; `event+0x0c` = event type.
- Jump table covers event types **1–16** (`lea eax,[r9-1]` then `cmp eax,0Fh`). Mouse→KBM cases are types **9–14**. Gamepad cases are types **2–5** (ButtonDown, ButtonUp, Axis, AnalogStick), now named in `keen` ([`notes/input-mode-events.md`](notes/input-mode-events.md)).
- Action size **280** bytes; header **8** bytes; KBM primary/secondary bindings at **+32** / **+84**.
- Directional slots in [`directional.go`](mod/mixedinput/directional.go): buttons `0`, gamepad analog `2`, mouse `4`, special `6`.
- Movement actions `0–3`, camera `4–7`.

### 4.2 Unknowns to resolve (early tasks)

1. Which jump-table event types store `InputModeGamepad`. **Resolved (1.1/1.2):** types 2–5. The Keyzen stick sends 5, and its buttons send 2/3.
2. Offsets of gamepad primary/secondary bindings within an action. **Resolved (1.3):** `+168` / `+220`.
3. How to call / locate the **gamepad** evaluate half (upstream already finds the KBM half relative to the active-store site). **Resolved (1.3):** call the same `evaluateGameplayActionHalf` with `action + 136` (the gamepad half, `r12`).

---

## 5. Constraints

1. **Do not modify** [`inspiration/`](inspiration/).
2. **GPL-3.0:** keep license + attribution if distributing.
3. Prefer **small, verifiable increments**; do not invert everything in one commit/PR-sized dump.
4. Do not install alongside the original EnshroudedMixedInput mod (conflicts on `dinput8.dll`).
5. Signature patterns break on game updates — keep signature tests green when `enshrouded.exe` is available.
6. Ask mode / plan mode: follow user mode; this SRS does not authorize skipping verification.

---

## 6. Work plan (incremental tasks)

Work **in order**. Each task has a clear verify step. Check boxes in this file as you complete them (or track in chat).

---

### Phase 0 — Baseline (confirm foundation)

#### Task 0.1 — Clean rebuild [Done]

**Do:** From repo root, with toolchain on PATH:

```powershell
just clean
just build
```

**Verify:**

- [x] `dist/EnshroudedMixInput.dll` exists
- [x] `dist/dinput8.dll` exists
- [x] No build errors

#### Task 0.2 — Optional signature smoke test [Done]

**Do:**

```powershell
$env:EMIM_GAME_DIR = "D:\SteamLibrary\steamapps\common\Enshrouded"
just test
```

**Verify:**

- [x] Signature / jump-table tests that need the exe pass or skip cleanly with a clear reason
- [x] No unexpected package test failures

#### Task 0.3 — In-game baseline (current upstream behavior) [Done]

**Do:** Remove any other `dinput8.dll` / old EMI DLL. `just deploy`. Launch game. Touch stick, then mouse, then keyboard. Read `EnshroudedMixInput.log`.

**Verify:**

- [x] Log shows mod loaded + `mixed_input` installed
- [x] Observe current behavior (likely: mouse OK in gamepad mode; keyboard still flips — document what you see)

**Observed (2026-11-09):** Log clean (`EnshroudedMixInput loaded`, `mod installed` / `mixed_input`). Controller/stick input works. Mixing controller + keyboard causes a ~1–2s stutter as the game switches input mode (upstream polarity; expected until Phase 2–3).

---

### Phase 1 — Reverse-engineer gamepad mode switch

Goal: know which events flip UI to gamepad so we can neutralize them.

#### Task 1.1 — Map `updateInputModeFromEvent` jump table [Done]

**Do:** Using the existing pattern finder / debugger / dump of jump-table entries (same helpers as `mouseEventJumpTableEntries` in [`mixed_input.go`](mod/mixedinput/mixed_input.go)):

- Locate jump table base and each entry for event types 1–16.
- Classify each entry: no-change, KeyboardMouse store, Gamepad store, or other.

**Verify:**

- [x] Written table in this SRS or a short `notes/input-mode-events.md`: event type → case
- [x] Mouse 9–14 still match “KeyboardMouse store” before any new patch (sanity)

**Result (2026-11-09):** see [`notes/input-mode-events.md`](notes/input-mode-events.md). Gamepad store: types 2–5 (ButtonDown, ButtonUp, Axis, AnalogStick; AnalogStick only past a movement threshold). KeyboardMouse store: 7–14 (KeyDown, KeyUp, mouse). Everything else is no change. Reproduce with `TestInputModeJumpTableMap`.

#### Task 1.2 — Confirm gamepad event types in-game or via RE [Done]

**Do:** Identify which types fire on XInput stick move / axis (Keyzen). Prefer static RE first; confirm with logging if needed (temporary log in a detour is OK in a later task).

**Verify:**

- [x] List of event type IDs that set `InputModeGamepad`
- [x] Constants added (or drafted) in `keen/input.go` with comments

**Result (2026-11-09):** Gamepad store is types 2 ButtonDown, 3 ButtonUp, 4 Axis, 5 AnalogStick. In-game, the Keyzen stick sends only 5 AnalogStick (it switches the mode only past the XInput deadzone of `0.2395`), and a Keyzen button sends 2/3. No Axis events were seen. Event layout is in [`notes/input-mode-events.md`](notes/input-mode-events.md). The temporary trace mod `mod/inputtrace` was removed after Task 2.1 was verified.

#### Task 1.3 — Document gamepad binding layout [Done]

**Do:** Find primary/secondary gamepad binding offsets inside the 280-byte action (mirror KBM 32/84). Note how `updateGameplayAction` selects the gamepad half (`r12` in pattern comments).

**Verify:**

- [x] `ActionGamepadPrimaryBinding` / `ActionGamepadSecondaryBinding` (names flexible) values written down
- [x] Confidence note: how they were found (disasm / RTTI / comparison)

**Result (2026-11-09):** gamepad half at action `+136` (`lea r12, [rsi+88h]`), gamepad primary / secondary bindings at `+168` / `+220`. Each half is 136 bytes and each binding 52. `getGameplayActionBinding` and `updateGameplayAction` agree independently. The mod reads the offsets at install (`readBindingLayout`), and `TestBindingLayout` asserts them against `keen/input.go`. An in-game dump showed movement and camera gamepad bindings = 8 GamepadStick at `+168`, and 0 Unbound at `+220`. Details are in [`notes/action-layout.md`](notes/action-layout.md).

---

### Phase 2 — Lock UI to KeyboardMouse

#### Task 2.1 — Patch gamepad jump-table entries to “no change” [Done]

**Do:** Extend (or replace) the mouse-only jump-table patch so **gamepad mode-changing** entries point at the same no-change/`ret` path. Decide whether mouse entries still need patching:

- Preferred for our goal: mode stays KBM; mouse already wants KBM, so mouse→KBM is fine (may leave mouse entries alone).
- Must neutralize gamepad→Gamepad entries.

**Verify:**

- [x] `just build` succeeds
- [x] Unit/signature test updated if jump-table expectations change
- [x] In-game: moving Keyzen stick does **not** switch UI to gamepad prompts
- [x] Log still shows successful install

**Result (2026-11-09):** `gamepadEventJumpTableEntries` replaces the upstream mouse patch. It checks that types 2–5 share one case that stores Gamepad, then points those four entries at the no-change `ret`. Mouse entries are left alone so mouse input switches to KBM natively. `TestGamepadEventJumpTableEntries` covers the patch, and `TestInputModeJumpTableMap` now asserts that types 2–5 are the Gamepad store. In-game with the trace mod: AnalogStick (full push) and ButtonDown/ButtonUp (a Keyzen key mapped to XInput A) all arrived in KeyboardMouse mode with zero mode changes; keyboard and mouse behave natively. As expected, stick and gamepad buttons do nothing until Phase 3. The trace mod was then removed.

#### Task 2.2 — Ensure mode starts / stays KBM [Done]

**Do:** If stick alone never sets gamepad but something else still flips, harden (e.g. force-store KBM on those paths, or patch remaining cases). Prefer minimal patches.

**Verify:**

- [x] Stick spam + mouse look + typing: UI remains KBM for a multi-minute session
- [x] No stuttering mode thrash

**Result (2026-11-09):** No hardening needed. Added a permanent guard instead: the `updateGameplayAction` detour logs `input mode is Gamepad although gamepad events are patched to keep KeyboardMouse` once if it ever sees Gamepad mode. Soak session of a few minutes (stick spam with full pushes, the Keyzen key mapped to XInput A, mouse look and clicks, menus, alt-tab out and back): prompts stayed KBM, no stutter, and the log held only the load and install lines, so the mode never left KeyboardMouse.

#### Task 2.3 — Keyboard + mouse native check [Done]

**Do:** With mode locked KBM, play with keyboard + mouse only (stick idle).

**Verify:**

- [x] Movement keys, inventory, chat/typing as applicable work
- [x] Mouse look and buttons work
- [x] G1 and G2 satisfied without stick

**Result (2026-11-09):** Verified in the same session as Task 2.2. Movement keys, menus, mouse look and buttons behave as in vanilla KBM. Chat typing was not explicitly exercised.

---

### Phase 3 — Merge gamepad stick into KBM mode

#### Task 3.1 — Invert `updateGameplayAction` merge [Done]

**Do:** Change detour so when mode == **KeyboardMouse**, also evaluate the **gamepad** half and OR into active (mirror current gamepad←KBM merge).

**Verify:**

- [x] Builds
- [x] With stick input, relevant actions become active while mode remains KBM (log/trace or in-game movement attempt)
- [x] Idle stick does not pin actions active

**Result (2026-11-09, code):** In KeyboardMouse mode the detour now calls `evaluateGameplayActionHalf(action + 136, index, time)` and sets active if it returns true, never clearing it. All gamepad bindings count, buttons included, so a Keyzen key mapped to XInput A should trigger Jump. The original already updates all four bindings (with their modifiers) before evaluating the mode's half, so the gamepad half is current. Gamepad mode now only logs the one-time warning; the old keyboard-mouse merge was removed. Each gamepad-caused activation that was not active the previous frame logs `gamepad half activated action` at trace level, using the previously active byte (`+0x111`, read at install). `TestActiveStoreOffsets` asserts active `+0x110` / previously active `+0x111`. `just build` and `just test` pass.

**Observed (2026-11-09, in-game with `EMIM_LOG_LEVEL=trace`):** UI stayed KBM with no stutter throughout. The Keyzen key mapped to XInput A makes the character jump. The stick does nothing visible, as expected until Task 3.2. The log confirms:

- The `index` argument of `updateGameplayAction` is the action number: action addresses are 280 bytes apart per index (index 73 = index 0 + 73 × 280).
- Stick pushes activate movement actions 0–3 in direction combinations (e.g. 1+2 = right+up). Camera actions 4–7 never fired, so the Keyzen stick reports as the left stick only.
- XInput A activates actions 73 and 78 together (one of them is Jump).
- Each activation logs once and the lines stop when idle, so no action is pinned active.
- No "input mode is Gamepad" warning.

#### Task 3.2 — Invert `addDirectionalActionInput` for stick/axis [Done]

**Do:** When mode == KBM, read gamepad stick/axis bindings and `addDirectional` into the gamepad analog slot. Focus first on **movement** actions 0–3.

**Verify:**

- [x] Builds
- [x] In-game: stick moves character with analog feel while UI stays KBM
- [x] Keyboard WASD still works (G1 + G3)

**Result (2026-11-09, code):** After the original adds the keyboard mouse value, the detour checks for KeyboardMouse mode, a movement action 0–3, and the action being active. It then reads the gamepad binding (`+168` primary / `+220` secondary, from `readBindingLayout`). Only GamepadStick (8) and GamepadAxis (7) kinds are merged, so gamepad buttons do not add to the keys. The value goes through `bindingAxisValue` and `addDirectional` into the gamepad analog slot (2/3). Camera actions are untouched (Task 3.4). The first non-zero stick value logs `gamepad stick added to movement input` once at debug level. The Gamepad-mode keyboard mouse merge was removed. `just build` and `just test` pass.

**Observed (2026-11-09, in-game):** The Keyzen stick moves the character with analog feel: a partial push walks slowly, a full push runs. So `computeMovementInput` does consume the gamepad analog slot in KeyboardMouse mode, and the contingency RE was not needed. WASD works on its own and together with the stick. Releasing the stick stops cleanly with no drift. XInput A still jumps. Prompts stayed KBM with no stutter.

#### Task 3.3 — Mouse look coexistence [Done]

**Do:** Confirm mouse relative look still works while stick is used for movement. Fix scaling/slot conflicts if look breaks.

**Verify:**

- [x] Can strafe with stick and look with mouse simultaneously
- [x] No sudden camera snaps from stick unless stick is bound to camera (out of scope to rematch gamepad camera unless easy)

**Observed (2026-11-09, in-game):** No code change needed. Mouse look behaves exactly as in vanilla while moving with the stick: no snaps, jerks or sensitivity change. This matches the code: camera actions 4–7 keep the game's own mouse path, and the stick merge only touches movement actions 0–3.

#### Task 3.4 — Decide camera-from-stick policy [Done]

**Do:** Explicit product choice: stick = movement only (recommended for Keyzen + mouse) vs also feed camera axes. Implement only the chosen policy; document in README.

**Verify:**

- [x] README states the policy
- [x] In-game behavior matches that policy

**Result (2026-11-09):** Policy is stick = movement only; the camera is the mouse's. No code change: the Task 3.2 merge already covers only movement actions 0–3. README has an "Input behavior" section stating this. In-game (Tasks 3.2 and 3.3) the stick moves the character and never turns the camera.

---

### Phase 4 — Cleanup and product polish

#### Task 4.1 — Remove or gate obsolete mouse-keep-gamepad behavior [Done]

**Do:** Upstream mouse jump-table patch may be unnecessary or harmful for our polarity. Remove dead code paths; rename mod (`mixed_input` → something clearer) if helpful.

**Verify:**

- [x] Code comments match actual behavior
- [x] `just build` + quick in-game smoke of G1–G4

**Result (2026-11-09, code):** The upstream mouse patch and both Gamepad-mode merges were already gone (Tasks 2.1, 3.1, 3.2). The remaining dead code was in [`directional.go`](mod/mixedinput/directional.go): the camera and target-select scaling of `bindingAxisValue` and the non-movement cases of `addDirectional` / `directionalSlot` were unreachable, since only movement actions 0–3 with stick/axis kinds are merged. They are replaced by `addGamepadAnalogMovement`, which adds the snapshot unscaled into slot 2/3, exactly what the old path computed for movement. The stale "upstream patch active" note in [`notes/input-mode-events.md`](notes/input-mode-events.md) is clarified. The mod keeps the name `mixed_input` (user decision). `just build` and `just test` pass.

**Observed (2026-11-09, in-game smoke):** All good after the cleanup: the stick walks or runs by push distance and stops cleanly, WASD works, mouse look is normal, XInput A jumps, and the UI stays KBM.

#### Task 4.2 — Update docs [Done]

**Do:** Update [`README.md`](README.md) status, goals, install notes, env vars. Keep GPL attribution. Optionally tick completed tasks in this SRS.

**Verify:**

- [x] README no longer says “behavior still upstream” once Phase 3 is done
- [x] SETUP still accurate

**Result (2026-11-09):** README status now says the mod works (keyboard + mouse + XInput stick together, UI stays KBM). Its "Input behavior" section (Task 3.4), install notes, env vars and GPL attribution are current. In SETUP the hardcoded `cd c:\MyStuff\...` path was removed; tools, versions (`go 1.27.2`) and `just` recipes match the repo.

#### Task 4.3 — Final acceptance pass [Done]

**Do:** Full play session with Keyzen stick + keyboard + mouse.

**Verify checklist:**

- [x] **G1** Keyboard works with stick in use
- [x] **G2** Mouse works with stick in use
- [x] **G3** Stick analog movement works
- [x] **G4** UI never switches to gamepad
- [x] Log clean of install errors
- [x] Uninstall (delete two DLLs) restores vanilla switching

**Observed (2026-11-09, in-game):** G1–G4 held across the Phase 3 and 4.1 sessions: WASD works alongside the stick, mouse look is unchanged, the stick walks or runs by push distance and stops cleanly, XInput A jumps, and prompts stay KBM with no stutter. The log of the cleaned-up build (`v0.0.0-cba5cfc`) shows only `EnshroudedMixInput loaded`, the hook line and `mod installed`, then trace/debug input lines, with no warnings or errors. The debug line `gamepad stick added to movement input ... kind=8` confirms the stick path. With both DLLs deleted, the mod writes no log (it is not loaded) and moving the stick brings back the vanilla mode-switch stutter. XInput A did not visibly stutter in vanilla, although RE shows ButtonDown also switches to Gamepad there.

---

## 7. Suggested chat workflow

1. Open a new chat; point at this file: `@srs.md`.
2. Say which task ID to implement (e.g. “Implement Task 1.1”).
3. Implement only that task; run its **Verify** steps before starting the next.
4. If RE blocks a task, stop and record findings under Phase 1 rather than guessing offsets.
5. Prefer Agent mode for code changes; Plan mode for multi-task design forks.

---

## 8. Quick reference — files you will touch

| Area | Paths |
|------|--------|
| Behavior | [`mod/mixedinput/mixed_input.go`](mod/mixedinput/mixed_input.go), [`directional.go`](mod/mixedinput/directional.go), [`patterns.go`](mod/mixedinput/patterns.go) |
| Constants | [`keen/input.go`](keen/input.go) |
| Tests | [`mod/mixedinput/jump_table_test.go`](mod/mixedinput/jump_table_test.go), [`signature_test.go`](mod/mixedinput/signature_test.go) |
| Loader name/env | [`loader/proxy.c`](loader/proxy.c) (only if rename needed) |
| Docs | [`README.md`](README.md), this [`srs.md`](srs.md) |
| Reference only | [`inspiration/`](inspiration/) |

---

## 9. Risk register

| Risk | Mitigation |
|------|------------|
| Wrong gamepad binding offsets | Phase 1.3 before Phase 3; verify with one action first |
| Missed gamepad event type still flips UI | Phase 1.1 full table; Phase 2.2 soak test |
| Game update breaks signatures | Keep pattern tests; re-scan with `EMI`/`EMIM_GAME_DIR` |
| Both mods installed | Document; deploy only our `dinput8.dll` |
| Stick deadzone / noise activates actions | Threshold / only merge stick+axis kinds, not all gamepad buttons |

---

## 10. Definition of done (project)

Project is done when Phase 4.3 verify checklist passes on the user’s Keyzen + mouse + keyboard setup, docs match behavior, and `inspiration/` remains an unmodified reference.
