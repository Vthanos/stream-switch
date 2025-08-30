PROTO=proto/streams.proto
PROTO_DIR=proto

.PHONY: deps proto run-backend run-gateway run-sim run-all

deps:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.34.2
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.4.0

proto:
	protoc -I $(PROTO_DIR) \
	  --go_out=$(PROTO_DIR) --go_opt=paths=source_relative \
	  --go-grpc_out=$(PROTO_DIR) --go-grpc_opt=paths=source_relative \
	  $(PROTO)

run-backend:
	go run ./backend/go/cmd/server

run-gateway:
	GRPC_TARGET=localhost:50051 go run ./gateway/ws/cmd/gateway

run-sim:
	GRPC_TARGET=localhost:50051 go run ./tools/sim/cmd/sim

run-all:
	@echo "Starting backend, gateway, and simulator... open http://localhost:8080"
	@bash -lc 'trap "exit" INT; \
	( go run ./backend/go/cmd/server ) & sleep 0.5; \
	( GRPC_TARGET=localhost:50051 go run ./gateway/ws/cmd/gateway ) & sleep 0.5; \
	( GRPC_TARGET=localhost:50051 go run ./tools/sim/cmd/sim ) & wait'
