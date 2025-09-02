package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	protobuf "p2p_client/grpc"
)

// P2PMessage represents a message structure used in P2P communication
type P2PMessage struct {
	MessageID    string // Unique identifier for the message
	Topic        string // Topic name where the message was published
	Message      []byte // Actual message data
	SourceNodeID string // ID of the node that sent the message (we don't need it in future, it is just for debug purposes)
}

// TraceData represents structured trace information for analysis
type TraceData struct {
	Event          string    `json:"event"`
	Timestamp      time.Time `json:"timestamp"`
	MessageID      string    `json:"message_id,omitempty"`
	Topic          string    `json:"topic,omitempty"`
	NodeID         string    `json:"node_id,omitempty"`
	LatencyMs      int       `json:"latency_ms,omitempty"`
	BandwidthBytes int       `json:"bandwidth_bytes,omitempty"`
	ShardID        string    `json:"shard_id,omitempty"`
	ShardIndex     int       `json:"shard_index,omitempty"`
	TotalShards    int       `json:"total_shards,omitempty"`
	Redundancy     float64   `json:"redundancy,omitempty"`
	Protocol       string    `json:"protocol,omitempty"`
}

// GossipSubTraceData represents GossipSub-specific trace information
type GossipSubTraceData struct {
	TraceData
	PeerID         string `json:"peer_id,omitempty"`
	MessageSize    int    `json:"message_size,omitempty"`
	DeliveryStatus string `json:"delivery_status,omitempty"`
	Hops           int    `json:"hops,omitempty"`
}

// OptimumP2PTraceData represents OptimumP2P-specific trace information
type OptimumP2PTraceData struct {
	TraceData
	CodedShards        int     `json:"coded_shards,omitempty"`
	ReceivedShards     int     `json:"received_shards,omitempty"`
	ReconstructionTime int     `json:"reconstruction_time_ms,omitempty"`
	Efficiency         float64 `json:"efficiency,omitempty"`
}

// Command possible operation that sidecar may perform with p2p node
type Command int32

const (
	CommandUnknown Command = iota
	CommandPublishData
	CommandSubscribeToTopic
	CommandUnSubscribeToTopic
)

var (
	addr    = flag.String("addr", "localhost:33212", "sidecar gRPC address")
	mode    = flag.String("mode", "subscribe", "mode: subscribe | publish")
	topic   = flag.String("topic", "", "topic name")
	message = flag.String("msg", "", "message data (for publish)")

	// optional: number of messages to publish (for stress testing or batch sending)
	count = flag.Int("count", 1, "number of messages to publish (for publish mode)")
	// optional: sleep duration between publishes
	sleep = flag.Duration("sleep", 0, "optional delay between publishes (e.g., 1s, 500ms)")
)

func main() {
	flag.Parse()
	if *topic == "" {
		log.Fatalf("−topic is required")
	}

	// connect with simple gRPC settings
	println(fmt.Sprintf("Connecting to node at: %s…", *addr))
	conn, err := grpc.NewClient(*addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(math.MaxInt),
			grpc.MaxCallSendMsgSize(math.MaxInt),
		),
	)
	if err != nil {
		log.Fatalf("failed to connect to node %v", err)
	}
	defer conn.Close()

	client := protobuf.NewCommandStreamClient(conn)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, err := client.ListenCommands(ctx)
	if err != nil {
		log.Fatalf("ListenCommands: %v", err)
	}

	// intercept CTRL+C for clean shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\nshutting down…")
		cancel()
		os.Exit(0)
	}()

	switch *mode {
	case "subscribe":
		println(fmt.Sprintf("Trying to subscribe to topic %s…", *topic))
		subReq := &protobuf.Request{
			Command: int32(CommandSubscribeToTopic),
			Topic:   *topic,
		}
		if err := stream.Send(subReq); err != nil {
			log.Fatalf("send subscribe: %v", err)
		}
		fmt.Printf("Subscribed to topic %q, waiting for messages…\n", *topic)

		var receivedCount int32
		msgChan := make(chan *protobuf.Response, 10000)

		// recv goroutine
		go func() {
			for {
				resp, err := stream.Recv()
				if err == io.EOF {
					close(msgChan)
					return
				}
				if err != nil {
					log.Printf("recv error: %v", err)
					close(msgChan)
					return
				}
				msgChan <- resp
			}
		}()

		// message handler loop
		for {
			select {
			case <-ctx.Done():
				log.Printf("Context canceled. Total messages received: %d", atomic.LoadInt32(&receivedCount))
				return
			case resp, ok := <-msgChan:
				if !ok {
					log.Printf("Stream closed. Total messages received: %d", atomic.LoadInt32(&receivedCount))
					return
				}
				go func(resp *protobuf.Response) {
					handleResponse(resp, &receivedCount)
				}(resp)
			}
		}

	case "publish":
		if *message == "" && *count == 1 {
			log.Fatalf("−msg is required in publish mode")
		}
		for i := 0; i < *count; i++ {
			var data []byte
			if *count == 1 {
				data = []byte(*message)
			} else {
				// generate secure random 4-byte hex
				randomBytes := make([]byte, 4)
				if _, err := rand.Read(randomBytes); err != nil {
					log.Fatalf("failed to generate random bytes: %v", err)
				}
				randomSuffix := hex.EncodeToString(randomBytes)
				data = []byte(fmt.Sprintf("P2P message %d - %s", i+1, randomSuffix))
			}

			pubReq := &protobuf.Request{
				Command: int32(CommandPublishData),
				Topic:   *topic,
				Data:    data,
			}
			if err := stream.Send(pubReq); err != nil {
				log.Fatalf("send publish: %v", err)
			}
			fmt.Printf("Published %q to %q\n", string(data), *topic)

			if *sleep > 0 {
				time.Sleep(*sleep)
			}
		}

	default:
		log.Fatalf("unknown mode %q", *mode)
	}
}

func handleResponse(resp *protobuf.Response, counter *int32) {
	switch resp.GetCommand() {
	case protobuf.ResponseType_Message:
		var p2pMessage P2PMessage
		if err := json.Unmarshal(resp.GetData(), &p2pMessage); err != nil {
			log.Printf("Error unmarshalling message: %v", err)
			return
		}
		n := atomic.AddInt32(counter, 1)
		fmt.Printf("[%d] Received message: %q\n", n, string(p2pMessage.Message))
	case protobuf.ResponseType_MessageTraceGossipSub:
		handleGossipSubTrace(resp.GetData())
	case protobuf.ResponseType_MessageTraceOptimumP2P:
		handleOptimumP2PTrace(resp.GetData())
	case protobuf.ResponseType_Unknown:
	default:
		log.Println("Unknown response command:", resp.GetCommand())
	}
}

// handleGossipSubTrace parses and displays GossipSub trace data
func handleGossipSubTrace(data []byte) {
	// The trace data is protobuf binary from libp2p-pubsub TraceEvent
	// For now, display the raw binary data as this contains valuable metrics
	// Future: Could unmarshal pb.TraceEvent if protobuf definitions are available
	fmt.Printf("[TRACE] GossipSub trace received (protobuf binary): %d bytes\n", len(data))

	// Try to parse as JSON for structured trace data (fallback/future compatibility)
	var gossipSubTrace GossipSubTraceData
	if err := json.Unmarshal(data, &gossipSubTrace); err == nil {
		fmt.Printf("[TRACE] GossipSub %s: peer=%s, latency=%dms, size=%d bytes, hops=%d\n",
			gossipSubTrace.Event, gossipSubTrace.PeerID, gossipSubTrace.LatencyMs,
			gossipSubTrace.MessageSize, gossipSubTrace.Hops)
	}
}

// handleOptimumP2PTrace parses and displays OptimumP2P trace data
func handleOptimumP2PTrace(data []byte) {
	// The trace data is protobuf binary from optimum-p2p TraceEvent
	// For now, display the raw binary data as this contains valuable metrics
	// Future: Could unmarshal optimum_pb.TraceEvent if protobuf definitions are available
	fmt.Printf("[TRACE] OptimumP2P trace received (protobuf binary): %d bytes\n", len(data))

	// Try to parse as JSON for structured trace data (fallback/future compatibility)
	var optimumTrace OptimumP2PTraceData
	if err := json.Unmarshal(data, &optimumTrace); err == nil {
		fmt.Printf("[TRACE] OptimumP2P %s: shard=%s (%d/%d), redundancy=%.2f, efficiency=%.2f, latency=%dms\n",
			optimumTrace.Event, optimumTrace.ShardID, optimumTrace.ReceivedShards,
			optimumTrace.CodedShards, optimumTrace.Redundancy, optimumTrace.Efficiency, optimumTrace.LatencyMs)
	}
}
