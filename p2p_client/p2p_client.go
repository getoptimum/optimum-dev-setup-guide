package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"

	protobuf "p2p_client/grpc/proto"
)

// P2PMessage represents a message structure used in P2P communication
type P2PMessage struct {
	MessageID    string // Unique identifier for the message
	Topic        string // Topic name where the message was published
	Message      []byte // Actual message data
	SourceNodeID string // ID of the node that sent the message (we don't need it in future, it is just for debug purposes)
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

	// Keepalive configuration flags
	keepaliveTime    = flag.Duration("keepalive-internal", 2*time.Minute, "gRPC keepalive ping interval")
	keepaliveTimeout = flag.Duration("keepalive-timeout", 20*time.Second, "gRPC keepalive ping timeout")
)

func main() {
	flag.Parse()
	if *topic == "" {
		log.Fatalf("−topic is required")
	}

	// connect with improved keepalive settings to avoid "too_many_pings" error
	conn, err := grpc.NewClient(*addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(math.MaxInt),
			grpc.MaxCallSendMsgSize(math.MaxInt),
		),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                *keepaliveTime,    // Configurable ping interval
			Timeout:             *keepaliveTimeout, // Configurable ping timeout
			PermitWithoutStream: true,              // Allow pings even without active streams
		}))
	if err != nil {
		log.Fatalf("failed to connect to node %v", err)
	}

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
		subReq := &protobuf.Request{
			Command: int32(CommandSubscribeToTopic),
			Topic:   *topic,
		}
		if err := stream.Send(subReq); err != nil {
			log.Fatalf("send subscribe: %v", err)
		}
		fmt.Printf("Subscribed to topic %q, waiting for messages…\n", *topic)
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				log.Println("stream closed by server")
				return
			}
			if err != nil {
				// Handle keepalive errors more gracefully
				if st, ok := status.FromError(err); ok {
                                     msg := st.Message()
		                     if strings.Contains(msg, "ENHANCE_YOUR_CALM") || strings.Contains(msg, "too_many_pings") {
						log.Printf("Connection closed due to keepalive ping limit. This indicates the server has stricter ping limits than expected.")
						log.Printf("Consider adjusting keepalive settings or server configuration.")
						return
					}
				}
				log.Fatalf("recv: %v", err)
			}
			handleResponse(resp)
		}

	case "publish":
		if *message == "" {
			log.Fatalf("−msg is required in publish mode")
		}
		pubReq := &protobuf.Request{
			Command: int32(CommandPublishData),
			Topic:   *topic,
			Data:    []byte(*message),
		}
		if err := stream.Send(pubReq); err != nil {
			log.Fatalf("send publish: %v", err)
		}
		// graceful wait for ACK or just sleep briefly
		fmt.Printf("Published %q to %q\n", *message, *topic)
		time.Sleep(500 * time.Millisecond)

	default:
		log.Fatalf("unknown mode %q", *mode)
	}
}

func handleResponse(resp *protobuf.Response) {
	switch resp.GetCommand() {
	case protobuf.ResponseType_Message:
		var p2pMessage P2PMessage
		if err := json.Unmarshal(resp.GetData(), &p2pMessage); err != nil {
			log.Fatalf("Error unmarshalling message %v", err)
			return
		}
		fmt.Printf("Received message: %q\n", string(p2pMessage.Message))
	case protobuf.ResponseType_MessageTraceGossipSub:
	case protobuf.ResponseType_MessageTraceOptimumP2P:
	case protobuf.ResponseType_Unknown:
	default:
		log.Println("Unknown response command:", resp.GetCommand())
	}
}
