package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
	pb "streamswitch/sensorv1"
)

//go:embed ../../../../ui/index.html
var indexHTML []byte

//go:embed ../../../../ui/script.js
var scriptJS []byte

var upgrader = websocket.Upgrader{ CheckOrigin: func(r *http.Request) bool { return true } }

func main() {
	target := os.Getenv("GRPC_TARGET")
	if target == "" { target = "localhost:50051" }

	conn, err := grpc.Dial(target, grpc.WithInsecure())
	if err != nil { log.Fatalf("dial gRPC target %s: %v", target, err) }
	defer conn.Close()
	client := pb.NewTelemetryClient(conn)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8"); w.Write(indexHTML)
	})
	http.HandleFunc("/script.js", func(w http.ResponseWriter, r *http.Request) {
	 w.Header().Set("Content-Type", "application/javascript; charset=utf-8"); w.Write(scriptJS)
	})

	http.HandleFunc("/ws/subscribe", func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil); if err != nil { return }
		defer ws.Close()
		sensor := r.URL.Query().Get("sensor_id")
		req := &pb.Subscription{}; if sensor != "" { req.SensorIds = []string{sensor} }
		ctx, cancel := context.WithCancel(r.Context()); defer cancel()
		stream, err := client.Subscribe(ctx, req)
		if err != nil { _ = ws.WriteMessage(websocket.TextMessage, []byte(`{"error":"subscribe failed"}`)); return }
		for {
			msg, err := stream.Recv(); if err != nil { return }
			type frame struct {
				Reading *pb.Reading      `json:"reading"`
				Meta    *pb.ServerMeta   `json:"meta"`
				ClientRecvUnixNano int64 `json:"client_recv_unix_nano"`
			}
			f := frame{ Reading: msg.Reading, Meta: msg.Meta, ClientRecvUnixNano: time.Now().UnixNano() }
			b, _ := json.Marshal(f)
			if err := ws.WriteMessage(websocket.TextMessage, b); err != nil { return }
		}
	})

	port := os.Getenv("PORT"); if port == "" { port = "8080" }
	log.Printf("WS Gateway serving UI on :%s (gRPC target %s)", port, target)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
