package main

import (
	"context"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	pb "streamswitch/sensorv1"
)

func main() {
	target := os.Getenv("GRPC_TARGET")
	if target == "" { target = "localhost:50051" }
	conn, err := grpc.Dial(target, grpc.WithInsecure())
	if err != nil { log.Fatal(err) }
	defer conn.Close()
	client := pb.NewTelemetryClient(conn)

	stream, err := client.Publish(context.Background())
	if err != nil { log.Fatal(err) }

	sensorIDs := []string{
		"sensor-"+uuid.New().String()[:8],
		"sensor-"+uuid.New().String()[:8],
		"sensor-"+uuid.New().String()[:8],
	}
	seq := uint64(0)
	ticker := time.NewTicker(50 * time.Millisecond); defer ticker.Stop()

	log.Printf("Simulator publishing to %s; sensors: %v", target, sensorIDs)
	for range ticker.C {
		seq++
		id := sensorIDs[rand.Intn(len(sensorIDs))]
		now := time.Now().UnixNano()
		r := &pb.Reading{
			SensorId:   id,
			TsUnixNano: now,
			Value:      20 + 10*rand.Float64(),
			Seq:        seq,
		}
		if err := stream.Send(r); err != nil { log.Printf("send error: %v", err); break }
	}
	ack, _ := stream.CloseAndRecv()
	log.Printf("closed, ack: %+v", ack)
}
