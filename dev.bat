@echo off
SET CMD=%1

IF "%CMD%"=="deps" (
  go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.34.2
  go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.4.0
  goto :eof
)

IF "%CMD%"=="proto" (
  protoc -I proto --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/streams.proto
  goto :eof
)

IF "%CMD%"=="run-backend" (
  go run backend/go/cmd/server
  goto :eof
)

IF "%CMD%"=="run-gateway" (
  set GRPC_TARGET=localhost:50051
  go run gateway/ws/cmd/gateway
  goto :eof
)

IF "%CMD%"=="run-sim" (
  set GRPC_TARGET=localhost:50051
  go run tools/sim/cmd/sim
  goto :eof
)

IF "%CMD%"=="run-all" (
  echo Starting backend, gateway, and simulator...
  start cmd /k "go run backend/go/cmd/server"
  timeout /t 1 >nul
  start cmd /k "set GRPC_TARGET=localhost:50051 && go run gateway/ws/cmd/gateway"
  timeout /t 1 >nul
  start cmd /k "set GRPC_TARGET=localhost:50051 && go run tools/sim/cmd/sim"
  echo Open http://localhost:8080
  goto :eof
)

echo Usage: dev.bat [deps|proto|run-backend|run-gateway|run-sim|run-all]
