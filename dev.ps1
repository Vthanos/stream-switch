param (
    [Parameter(Position=0)]
    [string]$cmd = "help"
)

switch ($cmd) {
    "deps" {
        go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.34.2
        go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.4.0
    }
    "proto" {
        protoc -I proto `
          --go_out=. --go_opt=paths=source_relative `
          --go-grpc_out=. --go-grpc_opt=paths=source_relative `
          proto/streams.proto
    }
    "run-backend" {
        go run backend/go/cmd/server
    }
    "run-gateway" {
        $env:GRPC_TARGET="localhost:50051"
        go run gateway/ws/cmd/gateway
    }
    "run-sim" {
        $env:GRPC_TARGET="localhost:50051"
        go run tools/sim/cmd/sim
    }
    "run-all" {
        Write-Output "Starting backend, gateway, and simulator..."
        Start-Process powershell -ArgumentList "go run backend/go/cmd/server"
        Start-Sleep -Seconds 1
        Start-Process powershell -ArgumentList "powershell -Command `$env:GRPC_TARGET='localhost:50051'; go run gateway/ws/cmd/gateway"
        Start-Sleep -Seconds 1
        Start-Process powershell -ArgumentList "powershell -Command `$env:GRPC_TARGET='localhost:50051'; go run tools/sim/cmd/sim"
        Write-Output "Open http://localhost:8080"
    }
    default {
        Write-Output "Usage: .\dev.ps1 [deps|proto|run-backend|run-gateway|run-sim|run-all]"
    }
}
