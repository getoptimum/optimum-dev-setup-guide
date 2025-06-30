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
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"

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
	count   = flag.Int("count", 1, "number of times to send the message (for publish mode)")

	// Keepalive configuration flags
	keepaliveTime    = flag.Duration("keepalive-interval", 2*time.Minute, "gRPC keepalive ping interval")
	keepaliveTimeout = flag.Duration("keepalive-timeout", 20*time.Second, "gRPC keepalive ping timeout")

	// Flow control configuration
	bufferSize = flag.Int("buffer-size", 1000, "message processing buffer size")
	workers    = flag.Int("workers", 4, "number of concurrent message processing workers")
)

// P2PClient manages the connection and provides message handling
type P2PClient struct {
	addr    string
	client  protobuf.CommandStreamClient
	conn    *grpc.ClientConn
	stream  protobuf.CommandStream_ListenCommandsClient
	ctx     context.Context
	cancel  context.CancelFunc
	topic   string
	mode    string
	message string
	count   int

	// Flow control and concurrency
	messageChan   chan *protobuf.Response
	receivedCount int64
	wg            sync.WaitGroup
	shutdownChan  chan struct{}

	// Statistics
	startTime       time.Time
	lastMessageTime time.Time
}

// NewP2PClient creates a new P2P client with flow control
func NewP2PClient(addr, mode, topic, message string, count int) *P2PClient {
	ctx, cancel := context.WithCancel(context.Background())
	return &P2PClient{
		addr:         addr,
		ctx:          ctx,
		cancel:       cancel,
		topic:        topic,
		mode:         mode,
		message:      message,
		count:        count,
		messageChan:  make(chan *protobuf.Response, *bufferSize),
		shutdownChan: make(chan struct{}),
		startTime:    time.Now(),
	}
}

// connect establishes a gRPC connection with optimized settings
func (c *P2PClient) connect() error {
	// Enhanced connection settings for flow control
	conn, err := grpc.NewClient(c.addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(math.MaxInt),
			grpc.MaxCallSendMsgSize(math.MaxInt),
		),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                *keepaliveTime,
			Timeout:             *keepaliveTimeout,
			PermitWithoutStream: false, // Keepalive fix from previous issue
		}),
		// Additional flow control settings
		grpc.WithInitialWindowSize(1024*1024),     // 1MB initial window
		grpc.WithInitialConnWindowSize(1024*1024), // 1MB connection window
	)
	if err != nil {
		return fmt.Errorf("failed to connect to node %s: %v", c.addr, err)
	}

	c.conn = conn
	c.client = protobuf.NewCommandStreamClient(conn)
	return nil
}

// establishStream creates the bidirectional stream
func (c *P2PClient) establishStream() error {
	stream, err := c.client.ListenCommands(c.ctx)
	if err != nil {
		return fmt.Errorf("ListenCommands failed: %v", err)
	}
	c.stream = stream
	return nil
}

// startMessageWorkers starts concurrent message processing workers
func (c *P2PClient) startMessageWorkers() {
	for i := 0; i < *workers; i++ {
		c.wg.Add(1)
		go c.messageWorker(i)
	}
}

// messageWorker processes messages from the channel
func (c *P2PClient) messageWorker(id int) {
	defer c.wg.Done()

	for {
		select {
		case resp := <-c.messageChan:
			if resp == nil {
				return // Shutdown signal
			}
			c.processMessage(resp)
		case <-c.shutdownChan:
			return
		}
	}
}

// processMessage handles a single message with proper error handling
func (c *P2PClient) processMessage(resp *protobuf.Response) {
	atomic.AddInt64(&c.receivedCount, 1)
	c.lastMessageTime = time.Now()

	switch resp.GetCommand() {
	case protobuf.ResponseType_Message:
		var p2pMessage P2PMessage
		if err := json.Unmarshal(resp.GetData(), &p2pMessage); err != nil {
			log.Printf("Error unmarshalling message: %v", err)
			return
		}
		fmt.Printf("Received message: %q\n", string(p2pMessage.Message))

		// Log progress every 100 messages
		if atomic.LoadInt64(&c.receivedCount)%100 == 0 {
			c.logProgress()
		}

	case protobuf.ResponseType_MessageTraceGossipSub:
		// Handle trace messages if needed
	case protobuf.ResponseType_MessageTraceOptimumP2P:
		// Handle trace messages if needed
	case protobuf.ResponseType_Unknown:
		log.Println("Unknown response command:", resp.GetCommand())
	default:
		log.Println("Unknown response command:", resp.GetCommand())
	}
}

// logProgress logs current statistics
func (c *P2PClient) logProgress() {
	elapsed := time.Since(c.startTime)
	rate := float64(atomic.LoadInt64(&c.receivedCount)) / elapsed.Seconds()
	fmt.Printf("[PROGRESS] Received: %d messages | Rate: %.2f msg/s | Elapsed: %s\n",
		atomic.LoadInt64(&c.receivedCount), rate, elapsed.Round(time.Second))
}

// startMessageReceiver runs the message receiving loop in a separate goroutine
func (c *P2PClient) startMessageReceiver() {
	go func() {
		defer close(c.messageChan)

		for {
			select {
			case <-c.ctx.Done():
				return
			case <-c.shutdownChan:
				return
			default:
				resp, err := c.stream.Recv()
				if err == io.EOF {
					log.Println("Stream closed by server")
					return
				}
				if err != nil {
					// Handle keepalive errors more gracefully
					if st, ok := status.FromError(err); ok {
						msg := st.Message()
						if strings.Contains(msg, "ENHANCE_YOUR_CALM") || strings.Contains(msg, "too_many_pings") {
							log.Printf("Connection closed due to keepalive ping limit.")
							return
						}
					}
					log.Printf("Stream receive error: %v", err)
					return
				}

				// Send message to processing workers (non-blocking)
				select {
				case c.messageChan <- resp:
					// Message sent successfully
				default:
					// Buffer full - log warning but continue
					log.Printf("WARNING: Message buffer full, dropping message")
				}
			}
		}
	}()
}

// subscribe sends the subscription request
func (c *P2PClient) subscribe() error {
	subReq := &protobuf.Request{
		Command: int32(CommandSubscribeToTopic),
		Topic:   c.topic,
	}
	if err := c.stream.Send(subReq); err != nil {
		return fmt.Errorf("send subscribe failed: %v", err)
	}
	return nil
}

// publish sends multiple publish requests with flow control
func (c *P2PClient) publish() error {
	pubReq := &protobuf.Request{
		Command: int32(CommandPublishData),
		Topic:   c.topic,
		Data:    []byte(c.message),
	}

	fmt.Printf("Publishing %d messages to topic %q...\n", c.count, c.topic)

	for i := 0; i < c.count; i++ {
		if err := c.stream.Send(pubReq); err != nil {
			return fmt.Errorf("send publish %d failed: %v", i+1, err)
		}

		if i%100 == 0 && i > 0 {
			fmt.Printf("Published %d messages...\n", i)
		}

		// Small delay to prevent overwhelming the server
		time.Sleep(10 * time.Millisecond)
	}

	fmt.Printf("Successfully published %d messages to %q\n", c.count, c.topic)
	return nil
}

// run executes the main client logic
func (c *P2PClient) run() error {
	switch c.mode {
	case "subscribe":
		if err := c.subscribe(); err != nil {
			return err
		}

		fmt.Printf("Subscribed to topic %q, waiting for messages...\n", c.topic)
		fmt.Printf("Buffer size: %d | Workers: %d\n", *bufferSize, *workers)

		// Start message processing infrastructure
		c.startMessageWorkers()
		c.startMessageReceiver()

		// Wait for shutdown signal
		<-c.shutdownChan

		// Log final statistics
		c.logFinalStats()

	case "publish":
		if c.message == "" {
			return fmt.Errorf("message is required in publish mode")
		}
		return c.publish()

	default:
		return fmt.Errorf("unknown mode %q", c.mode)
	}

	return nil
}

// logFinalStats logs final statistics
func (c *P2PClient) logFinalStats() {
	totalReceived := atomic.LoadInt64(&c.receivedCount)
	elapsed := time.Since(c.startTime)
	rate := float64(totalReceived) / elapsed.Seconds()

	fmt.Printf("\n=== FINAL STATISTICS ===\n")
	fmt.Printf("Total messages received: %d\n", totalReceived)
	fmt.Printf("Total time: %s\n", elapsed.Round(time.Second))
	fmt.Printf("Average rate: %.2f messages/second\n", rate)
	if totalReceived > 0 {
		fmt.Printf("Last message received: %s ago\n", time.Since(c.lastMessageTime).Round(time.Second))
	}
	fmt.Printf("=======================\n")
}

// Close cleans up the client resources
func (c *P2PClient) Close() {
	// Signal shutdown
	close(c.shutdownChan)

	// Cancel context
	if c.cancel != nil {
		c.cancel()
	}

	// Wait for workers to finish
	c.wg.Wait()

	// Close connection
	if c.conn != nil {
		c.conn.Close()
	}
}

func main() {
	flag.Parse()
	if *topic == "" {
		log.Fatalf("topic is required")
	}

	// Create client with flow control
	client := NewP2PClient(*addr, *mode, *topic, *message, *count)
	defer client.Close()

	// Handle graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\nShutting down gracefully...")
		client.Close()
		os.Exit(0)
	}()

	// Connect and run
	if err := client.connect(); err != nil {
		log.Fatalf("Connection failed: %v", err)
	}

	if err := client.establishStream(); err != nil {
		log.Fatalf("Stream establishment failed: %v", err)
	}

	if err := client.run(); err != nil {
		log.Fatalf("Client failed: %v", err)
	}
}
