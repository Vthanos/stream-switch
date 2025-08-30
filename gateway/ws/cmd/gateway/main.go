package main

import (
	"context"
	"embed"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "streamswitch/proto"
)

//go:embed web/*
var webFS embed.FS

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func resolveTarget() string {
	addr := flag.String("target", "", "gRPC target host:port (overrides GRPC_TARGET)")
	flag.Parse()

	target := *addr
	if target == "" {
		target = os.Getenv("GRPC_TARGET")
	}
	// sanitize: must contain ":" and NOT contain "/"
	if target == "" || strings.Contains(target, "/") || !strings.Contains(target, ":") {
		target = "localhost:50051"
	}
	return target
}

func main() {
	target := resolveTarget()
	log.Printf("[gateway] dialing gRPC target %q", target)

	conn, err := grpc.Dial(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("dial gRPC target %s: %v", target, err)
	}
	defer conn.Close()
	client := pb.NewTelemetryClient(conn)

	// Serve embedded UI: / -> web/index.html, /script.js -> web/script.js, etc.
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.ServeFileFS(w, r, webFS, "web/index.html")
			return
		}
		http.ServeFileFS(w, r, webFS, "web"+r.URL.Path)
	})

	// WebSocket endpoint: /ws/subscribe?sensor_id=*
	http.HandleFunc("/ws/subscribe", func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer ws.Close()

		sensor := r.URL.Query().Get("sensor_id")
		req := &pb.Subscription{}
		if sensor != "" {
			req.SensorIds = []string{sensor}
		}

		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		stream, err := client.Subscribe(ctx, req)
		if err != nil {
			_ = ws.WriteMessage(websocket.TextMessage, []byte(`{"error":"subscribe failed"}`))
			return
		}

		for {
			msg, err := stream.Recv()
			if err != nil {
				return
			}
			type frame struct {
				Reading            *pb.Reading    `json:"reading"`
				Meta               *pb.ServerMeta `json:"meta"`
				ClientRecvUnixNano int64          `json:"client_recv_unix_nano"`
			}
			f := frame{
				Reading:            msg.Reading,
				Meta:               msg.Meta,
				ClientRecvUnixNano: time.Now().UnixNano(),
			}
			b, _ := json.Marshal(f)
			if err := ws.WriteMessage(websocket.TextMessage, b); err != nil {
				return
			}
		}
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("[gateway] serving UI on :%s (WS /ws/subscribe)", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
