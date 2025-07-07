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

// FlowControlConfig holds flow control configuration
type FlowControlConfig struct {
	InitialCredits     int           // Initial credits for credit-based flow control
	CreditIncrement    int           // Credits to add when receiving acknowledgment
	MaxRetries         int           // Maximum number of retries for failed messages
	RetryDelay         time.Duration // Delay between retries
	PacingDelay        time.Duration // Delay between message sends for pacing
	BufferSize         int           // Size of the message buffer
	AckTimeout         time.Duration // Timeout for acknowledgments
	AdaptivePacing     bool          // Enable adaptive pacing based on network conditions
	MaxConcurrentSends int           // Maximum concurrent message sends
}

// FlowController manages flow control for message publishing
type FlowController struct {
	config    FlowControlConfig
	credits   int32
	mu        sync.RWMutex
	semaphore chan struct{}
	stats     *FlowStats
}

// FlowStats tracks flow control statistics
type FlowStats struct {
	MessagesSent    int64
	MessagesAcked   int64
	MessagesDropped int64
	MessagesRetried int64
	TotalLatency    time.Duration
	mu              sync.RWMutex
}

// Command possible operation that sidecar may perform with p2p node
type Command int32

const (
	CommandUnknown Command = iota
	CommandPublishData
	CommandSubscribeToTopic
	CommandUnSubscribeToTopic
	CommandFlowControlAck // New command for flow control acknowledgment
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

	// Keepalive configuration flags
	keepaliveTime    = flag.Duration("keepalive-interval", 2*time.Minute, "gRPC keepalive ping interval")
	keepaliveTimeout = flag.Duration("keepalive-timeout", 20*time.Second, "gRPC keepalive ping timeout")

	// Flow control flags
	enableFlowControl = flag.Bool("flow-control", true, "enable flow control mechanisms")
	initialCredits    = flag.Int("initial-credits", 100, "initial credits for flow control")
	creditIncrement   = flag.Int("credit-increment", 10, "credits to add on acknowledgment")
	maxRetries        = flag.Int("max-retries", 3, "maximum retries for failed messages")
	retryDelay        = flag.Duration("retry-delay", 100*time.Millisecond, "delay between retries")
	pacingDelay       = flag.Duration("pacing-delay", 50*time.Microsecond, "delay between message sends")
	bufferSize        = flag.Int("buffer-size", 1000, "size of message buffer")
	ackTimeout        = flag.Duration("ack-timeout", 5*time.Second, "timeout for acknowledgments")
	adaptivePacing    = flag.Bool("adaptive-pacing", true, "enable adaptive pacing")
	maxConcurrent     = flag.Int("max-concurrent", 10, "maximum concurrent message sends")
)

func main() {
	flag.Parse()
	if *topic == "" {
		log.Fatalf("−topic is required")
	}

	// Initialize flow control configuration
	flowConfig := FlowControlConfig{
		InitialCredits:     *initialCredits,
		CreditIncrement:    *creditIncrement,
		MaxRetries:         *maxRetries,
		RetryDelay:         *retryDelay,
		PacingDelay:        *pacingDelay,
		BufferSize:         *bufferSize,
		AckTimeout:         *ackTimeout,
		AdaptivePacing:     *adaptivePacing,
		MaxConcurrentSends: *maxConcurrent,
	}

	// connect with improved keepalive settings to avoid "too_many_pings" error
	conn, err := grpc.NewClient(*addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithInitialWindowSize(1024*1024*1024),
		grpc.WithInitialConnWindowSize(1024*1024*1024),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(math.MaxInt),
			grpc.MaxCallSendMsgSize(math.MaxInt),
		),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                *keepaliveTime,    // Configurable ping interval
			Timeout:             *keepaliveTimeout, // Configurable ping timeout
			PermitWithoutStream: false,             // Disable pings without active streams to avoid "too_many_pings" error
		}))
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

	// Initialize flow controller
	var flowController *FlowController
	if *enableFlowControl {
		flowController = NewFlowController(flowConfig)
		log.Printf("[FLOW CONTROL] Enabled with config: %+v", flowConfig)
	}

	// intercept CTRL+C for clean shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\nshutting down…")
		if flowController != nil {
			flowController.PrintStats()
		}
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
					if st, ok := status.FromError(err); ok {
						msg := st.Message()
						if strings.Contains(msg, "ENHANCE_YOUR_CALM") || strings.Contains(msg, "too_many_pings") {
							log.Printf("Connection closed due to keepalive ping limit. Server may have strict ping config.")
							close(msgChan)
							return
						}
					}
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
					handleResponse(resp, &receivedCount, flowController)
				}(resp)
			}
		}

	case "publish":
		if *message == "" && *count == 1 {
			log.Fatalf("−msg is required in publish mode")
		}

		if flowController != nil {
			// Use flow-controlled publishing
			publishWithFlowControl(ctx, stream, *topic, *message, *count, flowController)
		} else {
			// Use original publishing logic
			publishWithoutFlowControl(stream, *topic, *message, *count, *sleep)
		}

	default:
		log.Fatalf("unknown mode %q", *mode)
	}
}

// NewFlowController creates a new flow controller with the given configuration
func NewFlowController(config FlowControlConfig) *FlowController {
	return &FlowController{
		config:    config,
		credits:   int32(config.InitialCredits),
		semaphore: make(chan struct{}, config.MaxConcurrentSends),
		stats:     &FlowStats{},
	}
}

// publishWithFlowControl publishes messages with flow control mechanisms
func publishWithFlowControl(ctx context.Context, stream protobuf.CommandStream_ListenCommandsClient, topic, message string, count int, fc *FlowController) {
	log.Printf("[FLOW CONTROL] Starting flow-controlled publishing of %d messages", count)

	for i := 0; i < count; i++ {
		// Wait for credits
		if !fc.waitForCredits(ctx) {
			log.Printf("[FLOW CONTROL] No credits available, stopping publish")
			return
		}

		// Acquire semaphore for concurrent send limit
		select {
		case fc.semaphore <- struct{}{}:
		case <-ctx.Done():
			return
		}

		// Generate message data
		var data []byte
		if count == 1 {
			data = []byte(message)
		} else {
			randomBytes := make([]byte, 4)
			if _, err := rand.Read(randomBytes); err != nil {
				log.Fatalf("failed to generate random bytes: %v", err)
			}
			randomSuffix := hex.EncodeToString(randomBytes)
			data = []byte(fmt.Sprintf("P2P message %d - %s", i+1, randomSuffix))
		}

		// Send message with retry logic
		go func(msgData []byte, msgIndex int) {
			defer func() { <-fc.semaphore }()
			fc.sendWithRetry(ctx, stream, topic, msgData, msgIndex)
		}(data, i)

		// Apply pacing delay
		if fc.config.PacingDelay > 0 {
			time.Sleep(fc.config.PacingDelay)
		}
	}

	// Wait for all messages to be processed
	log.Printf("[FLOW CONTROL] Waiting for all messages to complete...")
	time.Sleep(2 * time.Second)
}

// publishWithoutFlowControl uses the original publishing logic
func publishWithoutFlowControl(stream protobuf.CommandStream_ListenCommandsClient, topic, message string, count int, sleep time.Duration) {
	for i := 0; i < count; i++ {
		var data []byte
		if count == 1 {
			data = []byte(message)
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
			Topic:   topic,
			Data:    data,
		}
		if err := stream.Send(pubReq); err != nil {
			log.Fatalf("send publish: %v", err)
		}
		fmt.Printf("Published %q to %q\n", string(data), topic)

		if sleep > 0 {
			time.Sleep(sleep)
		}
	}
}

// waitForCredits waits for available credits
func (fc *FlowController) waitForCredits(ctx context.Context) bool {
	for {
		fc.mu.RLock()
		credits := fc.credits
		fc.mu.RUnlock()

		if credits > 0 {
			return true
		}

		select {
		case <-ctx.Done():
			return false
		case <-time.After(100 * time.Millisecond):
			// Continue waiting
		}
	}
}

// sendWithRetry sends a message with retry logic
func (fc *FlowController) sendWithRetry(ctx context.Context, stream protobuf.CommandStream_ListenCommandsClient, topic string, data []byte, msgIndex int) {
	startTime := time.Now()

	for attempt := 0; attempt <= fc.config.MaxRetries; attempt++ {
		// Decrement credits
		fc.mu.Lock()
		if fc.credits <= 0 {
			fc.mu.Unlock()
			log.Printf("[FLOW CONTROL] No credits available for message %d", msgIndex)
			return
		}
		fc.credits--
		fc.mu.Unlock()

		// Send message
		pubReq := &protobuf.Request{
			Command: int32(CommandPublishData),
			Topic:   topic,
			Data:    data,
		}

		if err := stream.Send(pubReq); err != nil {
			log.Printf("[FLOW CONTROL] Send error for message %d (attempt %d): %v", msgIndex, attempt+1, err)

			// Restore credits on error
			fc.mu.Lock()
			fc.credits++
			fc.mu.Unlock()

			if attempt < fc.config.MaxRetries {
				time.Sleep(fc.config.RetryDelay)
				continue
			} else {
				fc.stats.mu.Lock()
				fc.stats.MessagesDropped++
				fc.stats.mu.Unlock()
				log.Printf("[FLOW CONTROL] Message %d dropped after %d retries", msgIndex, fc.config.MaxRetries)
				return
			}
		}

		// Message sent successfully - add credits back immediately since we don't have acks
		fc.mu.Lock()
		fc.credits += int32(fc.config.CreditIncrement)
		fc.mu.Unlock()

		fc.stats.mu.Lock()
		fc.stats.MessagesSent++
		fc.stats.MessagesAcked++ // Count as acked since we don't have real acks
		fc.stats.TotalLatency += time.Since(startTime)
		fc.stats.mu.Unlock()

		log.Printf("[FLOW CONTROL] Published message %d to %q (attempt %d)", msgIndex, topic, attempt+1)
		return
	}
}

// addCredits adds credits to the flow controller
func (fc *FlowController) addCredits(amount int) {
	fc.mu.Lock()
	fc.credits += int32(amount)
	fc.mu.Unlock()
}

// PrintStats prints flow control statistics
func (fc *FlowController) PrintStats() {
	fc.stats.mu.RLock()
	defer fc.stats.mu.RUnlock()

	log.Printf("[FLOW CONTROL STATS] Sent: %d, Acked: %d, Dropped: %d, Retried: %d",
		fc.stats.MessagesSent, fc.stats.MessagesAcked, fc.stats.MessagesDropped, fc.stats.MessagesRetried)

	if fc.stats.MessagesSent > 0 {
		avgLatency := fc.stats.TotalLatency / time.Duration(fc.stats.MessagesSent)
		log.Printf("[FLOW CONTROL STATS] Average latency: %v", avgLatency)
	}
}

func handleResponse(resp *protobuf.Response, counter *int32, fc *FlowController) {
	switch resp.GetCommand() {
	case protobuf.ResponseType_Message:
		var p2pMessage P2PMessage
		if err := json.Unmarshal(resp.GetData(), &p2pMessage); err != nil {
			log.Printf("Error unmarshalling message: %v", err)
			return
		}
		n := atomic.AddInt32(counter, 1)
		fmt.Printf("[%d] Received message: %q\n", n, string(p2pMessage.Message))

		// Add credits for flow control if enabled
		if fc != nil {
			fc.addCredits(fc.config.CreditIncrement)
			fc.stats.mu.Lock()
			fc.stats.MessagesAcked++
			fc.stats.mu.Unlock()
		}
	case protobuf.ResponseType_MessageTraceGossipSub:
	case protobuf.ResponseType_MessageTraceOptimumP2P:
	case protobuf.ResponseType_Unknown:
	default:
		log.Println("Unknown response command:", resp.GetCommand())
	}
}
