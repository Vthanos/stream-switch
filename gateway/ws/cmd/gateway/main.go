package main

import (
	"context"
	"embed"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
	pb "streamswitch/proto"
)

//go:embed web/*
var webFS embed.FS

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func main() {
	target := os.Getenv("GRPC_TARGET")
	if target == "" {
		target = "localhost:50051"
	}

	conn, err := grpc.Dial(target, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("dial gRPC target %s: %v", target, err)
	}
	defer conn.Close()
	client := pb.NewTelemetryClient(conn)

	// Serve embedded files
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.ServeFileFS(w, r, webFS, "web/index.html")
			return
		}
		// allow /script.js and any other static under web/
		http.ServeFileFS(w, r, webFS, "web"+r.URL.Path)
	})

	// WS endpoint
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
	log.Printf("WS Gateway serving UI on :%s (gRPC target %s)", port, target)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
