package main

import (
	"context"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"
	pb "streamswitch/proto"
)

type hub struct {
	mu   sync.RWMutex
	subs map[string][]chan *pb.ReadingWithMeta // key "*" or sensor_id
}

func newHub() *hub { return &hub{subs: make(map[string][]chan *pb.ReadingWithMeta)} }

func (h *hub) add(id string, ch chan *pb.ReadingWithMeta) func() {
	h.mu.Lock()
	h.subs[id] = append(h.subs[id], ch)
	h.mu.Unlock()
	return func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		arr := h.subs[id]
		for i := range arr {
			if arr[i] == ch {
				h.subs[id] = append(arr[:i], arr[i+1:]...)
				break
			}
		}
	}
}

func (h *hub) broadcast(r *pb.Reading) {
	now := time.Now().UnixNano()
	msg := &pb.ReadingWithMeta{
		Reading: r,
		Meta:    &pb.ServerMeta{ReceivedUnixNano: now, SentUnixNano: now},
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	if arr := h.subs["*"]; len(arr) > 0 {
		for _, ch := range arr {
			select {
			case ch <- msg:
			default:
			}
		}
	}
	if arr := h.subs[r.SensorId]; len(arr) > 0 {
		for _, ch := range arr {
			select {
			case ch <- msg:
			default:
			}
		}
	}
}

type telemetry struct {
	pb.UnimplementedTelemetryServer
	h *hub
}

func (t *telemetry) Publish(stream pb.Telemetry_PublishServer) error {
	var last uint64
	for {
		r, err := stream.Recv()
		if err != nil {
			return stream.SendAndClose(&pb.Ack{LastSeq: last})
		}
		last = r.Seq
		t.h.broadcast(r)
	}
}

func (t *telemetry) Subscribe(req *pb.Subscription, stream pb.Telemetry_SubscribeServer) error {
	ids := req.SensorIds
	if len(ids) == 0 {
		ids = []string{"*"}
	}
	ch := make(chan *pb.ReadingWithMeta, 1024)
	cleanups := make([]func(), 0, len(ids))
	for _, id := range ids {
		cleanups = append(cleanups, t.h.add(id, ch))
	}
	defer func() {
		for _, c := range cleanups {
			c()
		}
	}()

	sampleEvery := time.Duration(0)
	if req.SampleRateHz > 0 {
		sampleEvery = time.Second / time.Duration(req.SampleRateHz)
	}
	tick := time.NewTicker(time.Millisecond * 10)
	defer tick.Stop()
	var lastSend time.Time

	for {
		select {
		case m := <-ch:
			now := time.Now().UnixNano()
			m.Meta.SentUnixNano = now
			if sampleEvery > 0 {
				if time.Since(lastSend) < sampleEvery {
					continue
				}
				lastSend = time.Now()
			}
			if err := stream.Send(m); err != nil {
				return err
			}
		case <-tick.C:
		case <-stream.Context().Done():
			return nil
		}
	}
}

// Bench service
// Bench service
type bench struct{ pb.UnimplementedBenchServer }

func (b *bench) Ping(ctx context.Context, p *pb.PingRequest) (*pb.PingReply, error) {
	return &pb.PingReply{N: p.N, TsUnixNano: time.Now().UnixNano()}, nil
}

func (b *bench) StreamPing(s pb.Bench_StreamPingServer) error {
	for {
		msg, err := s.Recv()
		if err != nil {
			return err
		}
		reply := &pb.PingReply{N: msg.N, TsUnixNano: time.Now().UnixNano()}
		if err := s.Send(reply); err != nil {
			return err
		}
	}
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatal(err)
	}
	s := grpc.NewServer()
	h := newHub()
	pb.RegisterTelemetryServer(s, &telemetry{h: h})
	pb.RegisterBenchServer(s, &bench{})
	log.Println("Go gRPC backend listening on :50051")
	if err := s.Serve(lis); err != nil {
		log.Fatal(err)
	}
	_ = rand.Int()
}
