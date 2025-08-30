package main

import (
	"context"
	"flag"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "streamswitch/proto"
)

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
	log.Printf("[sim] dialing gRPC target %q", target)

	conn, err := grpc.Dial(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	client := pb.NewTelemetryClient(conn)
	stream, err := client.Publish(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	sensorIDs := []string{
		"sensor-" + uuid.New().String()[:8],
		"sensor-" + uuid.New().String()[:8],
		"sensor-" + uuid.New().String()[:8],
	}
	seq := uint64(0)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	log.Printf("[sim] publishing ~20 msg/s across %d sensors", len(sensorIDs))
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
		if err := stream.Send(r); err != nil {
			log.Printf("[sim] send error: %v", err)
			break
		}
	}
	ack, _ := stream.CloseAndRecv()
	log.Printf("[sim] closed, ack: %+v", ack)
}
