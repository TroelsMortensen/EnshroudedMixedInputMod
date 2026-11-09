# Build toolchain setup (Windows)

Required to build this project from the repo root:

| Tool | Role |
|------|------|
| **Go** (version from `go.mod`, currently 1.27.x) | Builds the mod DLL (`-buildmode=c-shared`) |
| **LLVM-MinGW UCRT** | C compiler for cgo (`CGO_ENABLED=1`) |
| **CMake** | Builds the `dinput8.dll` proxy loader |
| **just** | Task runner (`just build`, `just deploy`) |
| **goversioninfo** | Embeds Windows version resources into the Go DLL |

Optional: Git (version strings), golangci-lint (`just lint`), MSVC (not required; CI-style builds use LLVM-MinGW).

---

## 1. Go

1. Install from https://go.dev/dl/ (match or exceed `go` in `go.mod`).
2. Ensure `C:\Program Files\Go\bin` is on PATH.
3. Open a **new** terminal and verify:

```powershell
go version
where.exe go
```

If `go` is missing in Cursor but works in a fresh `cmd`, restart Cursor so terminals pick up PATH.

---

## 2. LLVM-MinGW (UCRT)

```powershell
winget install -e --id MartinStorsjo.LLVM-MinGW.UCRT
```

New terminal:

```powershell
gcc --version
```

Expected: Clang reporting `Target: x86_64-w64-windows-gnu` (LLVM-MinGW’s `gcc` is Clang).

---

## 3. CMake

```powershell
winget install -e --id Kitware.CMake
```

```powershell
cmake --version
```

---

## 4. just

```powershell
winget install -e --id Casey.Just
```

```powershell
just --version
```

---

## 5. goversioninfo

```powershell
go install github.com/josephspurrier/goversioninfo/cmd/goversioninfo@v1.7.0
```

Ensure `%USERPROFILE%\go\bin` is on PATH. New terminal:

```powershell
goversioninfo -h
```

---

## Verify + first build

In a **new** PowerShell (or after restarting Cursor):

```powershell
go version
gcc --version
cmake --version
just --version
goversioninfo -h
```

Then from the repo root:

```powershell
just build          # Debug → dist/
# or
just build Release
```

Outputs: `dist/EnshroudedMixInput.dll` and `dist/dinput8.dll`.

`just deploy` copies them into the game folder (`EMIM_GAME_DIR`, default Steam Enshrouded path).

---

## Notes

- After each `winget` / Go install, open a **new** terminal (or restart the IDE) so PATH updates apply.
- You do not need a dedicated Go IDE; Cursor/VS Code + the Go extension is enough.
- Debuggers for reversing (x64dbg, Cheat Engine, etc.) are separate from this build toolchain.
- `inspiration/` is read-only reference source; build this repo’s root project, not the inspiration tree.
