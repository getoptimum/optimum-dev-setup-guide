package main

import (
	"bytes"
	"context"
	crand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"time"

	protobuf "gateway_client/grpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

const (
	gatewayREST      = "http://localhost:8081"
	gatewayGRPC      = "localhost:50051"
	defaultTopic     = "demo"
	defaultThreshold = 0.1
	defaultMsgCount  = 5
	defaultDelay     = 2 * time.Second
)

var (
	topic         = flag.String("topic", defaultTopic, "topic name")
	threshold     = flag.Float64("threshold", defaultThreshold, "delivery threshold (0.0 to 1.0)")
	subscribeOnly = flag.Bool("subscribeOnly", false, "only subscribe and receive messages (no publishing)")
	messageCount  = flag.Int("count", defaultMsgCount, "number of messages to publish")
	messageDelay  = flag.Duration("delay", defaultDelay, "delay between message publishing")

	keepaliveTime    = flag.Duration("keepalive-interval", 2*time.Minute, "gRPC keepalive interval")
	keepaliveTimeout = flag.Duration("keepalive-timeout", 20*time.Second, "gRPC keepalive timeout")

	words = []string{"hello", "ping", "update", "broadcast", "status", "message", "event", "data", "note"}
)

func main() {
	flag.Parse()

	clientID := generateClientID()
	log.Printf("[INFO] Client ID: %s | Topic: %s | Threshold: %.2f", clientID, *topic, *threshold)

	// Subscribe via REST
	if err := subscribe(clientID, *topic, *threshold); err != nil {
		log.Fatalf("subscribe error: %v", err)
	}

	// Connect to gRPC stream
	conn, err := grpc.NewClient(gatewayGRPC,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(math.MaxInt),
			grpc.MaxCallSendMsgSize(math.MaxInt),
		),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                *keepaliveTime,
			Timeout:             *keepaliveTimeout,
			PermitWithoutStream: false,
		}),
	)
	if err != nil {
		log.Fatalf("gRPC connection failed: %v", err)
	}
	defer conn.Close()

	client := protobuf.NewGatewayStreamClient(conn)
	stream, err := client.ClientStream(context.Background())
	if err != nil {
		log.Fatalf("stream open failed: %v", err)
	}

	if err := stream.Send(&protobuf.GatewayMessage{ClientId: clientID}); err != nil {
		log.Fatalf("client ID send failed: %v", err)
	}

	// Handle incoming messages
	go func() {
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				log.Println("[CLOSED] gRPC stream closed by server")
				return
			}
			if err != nil {
				log.Printf("[ERROR] stream receive: %v", err)
				return
			}
			log.Printf("[RECEIVED] Topic: %s | Message: %s", resp.Topic, string(resp.Message))
		}
	}()

	// Trap SIGINT
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		log.Println("[INTERRUPTED] shutting down...")
		os.Exit(0)
	}()

	if *subscribeOnly {
		select {}
	}

	// Message publishing loop
	for i := 0; i < *messageCount; i++ {
		msg := generateRandomMessage()
		log.Printf("[PUBLISH] Message: %s", msg)
		if err := publishMessage(clientID, *topic, msg); err != nil {
			log.Printf("[ERROR] publish failed: %v", err)
		}
		time.Sleep(*messageDelay)
	}

	time.Sleep(3 * time.Second)
}

// subscribe registers the client with the Gateway via REST API
func subscribe(clientID, topic string, threshold float64) error {
	body := map[string]interface{}{
		"client_id": clientID,
		"topic":     topic,
		"threshold": threshold,
	}
	data, _ := json.Marshal(body)
	resp, err := http.Post(gatewayREST+"/api/subscribe", "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return nil
}

// publishMessage sends a REST request to publish a message
func publishMessage(clientID, topic, msg string) error {
	body := map[string]interface{}{
		"client_id": clientID,
		"topic":     topic,
		"message":   msg,
	}
	data, _ := json.Marshal(body)
	resp, err := http.Post(gatewayREST+"/api/publish", "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return nil
}

// generateClientID returns a random client identifier
func generateClientID() string {
	b := make([]byte, 4)
	_, _ = crand.Read(b)
	return "client_" + hex.EncodeToString(b)
}

// generateRandomMessage creates a random message with timestamp
func generateRandomMessage() string {
	return fmt.Sprintf("%s @ %s", words[rand.Intn(len(words))], time.Now().Format("15:04:05"))
}
