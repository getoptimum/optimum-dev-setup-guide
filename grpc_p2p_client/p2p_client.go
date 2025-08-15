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
		fmt.Printf("[TRACE] GossipSub trace received: %s\n", string(resp.GetData()))
	case protobuf.ResponseType_MessageTraceOptimumP2P:
		fmt.Printf("[TRACE] OptimumP2P trace received: %s\n", string(resp.GetData()))
	case protobuf.ResponseType_Unknown:
	default:
		log.Println("Unknown response command:", resp.GetCommand())
	}
}
