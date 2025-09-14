# StreamSwitch (Go 1.25, local module)

Visual demo + mini bench: **Go gRPC backend** → **WebSocket gateway** → **browser UI**.
Local module path, no GitHub needed.

## Quickstart (Windows)
1) Install Go 1.25, `protoc`, and ensure `~/go/bin` (or `%USERPROFILE%\go\bin`) is on PATH.
2) In PowerShell:
```
.\dev.ps1 deps
.\dev.ps1 proto
.\dev.ps1 run-all
```
Open http://localhost:8080

## Layout
- proto/streams.proto
- backend/go/cmd/server/main.go
- gateway/ws/cmd/gateway/main.go (serves UI via go:embed)
- tools/sim/cmd/sim/main.go
- ui/index.html, ui/script.js
- go.mod, tools.go
- dev.ps1 (PowerShell), dev.bat (CMD)
- Makefile (optional for Linux/macOS)

 change `module` in go.mod and `go_package` in proto accordingly and regenerate with `.\dev.ps1 proto`.
