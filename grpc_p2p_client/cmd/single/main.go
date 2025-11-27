package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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

	protobuf "p2p_client/grpc"
	"p2p_client/shared"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	addr    = flag.String("addr", "localhost:33212", "sidecar gRPC address")
	mode    = flag.String("mode", "subscribe", "mode: subscribe | publish")
	topic   = flag.String("topic", "", "topic name")
	message = flag.String("msg", "", "message data (for publish)")
	count   = flag.Int("count", 1, "number of messages to publish (for publish mode)")
	sleep   = flag.Duration("sleep", 0, "optional delay between publishes (e.g., 1s, 500ms)")
)

func main() {
	flag.Parse()
	if *topic == "" {
		log.Fatalf("−topic is required")
	}

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
		subscribe(ctx, stream, *topic)
	case "publish":
		publish(ctx, stream, *topic, *message, *count, *sleep)
	default:
		log.Fatalf("unknown mode %q", *mode)
	}
}

func subscribe(ctx context.Context, stream protobuf.CommandStream_ListenCommandsClient, topic string) {
	println(fmt.Sprintf("Trying to subscribe to topic %s…", topic))
	subReq := &protobuf.Request{
		Command: int32(shared.CommandSubscribeToTopic),
		Topic:   topic,
	}
	if err := stream.Send(subReq); err != nil {
		log.Fatalf("send subscribe: %v", err)
	}
	fmt.Printf("Subscribed to topic %q, waiting for messages…\n", topic)

	var receivedCount int32
	msgChan := make(chan *protobuf.Response, 10000)

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
				shared.HandleResponse(resp, &receivedCount)
			}(resp)
		}
	}
}

func publish(ctx context.Context, stream protobuf.CommandStream_ListenCommandsClient,
	topic, msg string, count int, sleep time.Duration) {

	if msg == "" && count == 1 {
		log.Fatalf("−msg is required in publish mode")
	}

	for i := 0; i < count; i++ {
		start := time.Now()
		var data []byte
		currentTime := time.Now().UnixNano()

		if count == 1 {
			prefix := fmt.Sprintf("[%d %d] ", currentTime, len(msg))
			prefixBytes := []byte(prefix)
			data = append(prefixBytes, msg...)
		} else {
			randomBytes := make([]byte, 4)
			if _, err := rand.Read(randomBytes); err != nil {
				log.Fatalf("failed to generate random bytes: %v", err)
			}
			randomSuffix := hex.EncodeToString(randomBytes)
			data = []byte(fmt.Sprintf("[%d %d] %d - %s XXX", currentTime, len(randomSuffix), i+1, randomSuffix))
		}

		pubReq := &protobuf.Request{
			Command: int32(shared.CommandPublishData),
			Topic:   topic,
			Data:    data,
		}
		if err := stream.Send(pubReq); err != nil {
			log.Fatalf("send publish: %v", err)
		}

		elapsed := time.Since(start)
		fmt.Printf("Published %q to %q (took %v)\n", string(data), topic, elapsed)

		if sleep > 0 {
			time.Sleep(sleep)
		}
	}
}
