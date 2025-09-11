package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
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
        "regexp"

	protobuf "p2p_client/grpc"
	optsub "p2p_client/grpc/mump2p_trace"

	"github.com/gogo/protobuf/proto"
	pubsubpb "github.com/libp2p/go-libp2p-pubsub/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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
			currentTime := time.Now().UnixNano()
	                sender_addr_re := regexp.MustCompile(`\d+\.\d+\.\d+\.\d+`)
	                sender_addr_info := sender_addr_re.FindString(*addr)

			if *count == 1 {
				// Create the prefix string and convert it to bytes
				approx_info_prefix := fmt.Sprintf("sender_addr:%s\t[send_time, size]:[%d, %d]\t", sender_addr_info, currentTime, len(*message))
                                correct_size := len(approx_info_prefix) + len(*message) +  1  

				prefix := fmt.Sprintf("sender_addr:%s\t[send_time, size]:[%d, %d]\t", sender_addr_info, currentTime, correct_size)
				prefixBytes := []byte(prefix)

				// Prepend the prefixBytes to your existing data
				data = append(prefixBytes, *message...)
			} else {
				// generate secure random 4-byte hex
				randomBytes := make([]byte, 4)
				if _, err := rand.Read(randomBytes); err != nil {
					log.Fatalf("failed to generate random bytes: %v", err)
				}
			//	randomSuffix := hex.EncodeToString(randomBytes)
			//	data = []byte(fmt.Sprintf("[%d %d] %d - %s XXX", currentTime, len(randomSuffix), i+1, randomSuffix))
			}

			pubReq := &protobuf.Request{
				Command: int32(CommandPublishData),
				Topic:   *topic,
				Data:    data,
			}
			if err := stream.Send(pubReq); err != nil {
				log.Fatalf("send publish: %v", err)
			}

                        sum := sha256.Sum256(data)   // returns []byte
                        hash := hex.EncodeToString(sum[:])   // returns [32]byte

			fmt.Printf("Publish:\tsender_info:%s, [send_time, size]:[%d, %d]\ttopic:%s\tmsg_hash:%s\n", sender_addr_info, currentTime, len(data), *topic,  string(hash)[:8])

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

		currentTime := time.Now().UnixNano()

		messageSize := len(p2pMessage.Message)
                sum := sha256.Sum256(p2pMessage.Message)   // returns []byte
                hash := hex.EncodeToString(sum[:])   // returns [32]byte

	        //send_info_re := regexp.MustCompile(`sender_addr:\d+\.\d+\.\d+\.\d+,\s\[send_time, size]:[%d, %d]`)
	        send_info_re := regexp.MustCompile(`sender_addr:\d+\.\d+\.\d+\.\d+\t\[send_time, size]:\[\d+,\s\d+\]`)
	        // Find the first match
	        send_info := send_info_re.FindString(string(p2pMessage.Message))

	        recv_addr_re := regexp.MustCompile(`\d+\.\d+\.\d+\.\d+`)
	        recv_addr_info := recv_addr_re.FindString(string(p2pMessage.Message))

		fmt.Printf("Recv:\t[%d]\treceiver_addr:%s\t[recv_time, size]:[%d, %d]\t%s\ttopic:%s\thash:%s\n", n, recv_addr_info, currentTime, messageSize, send_info, p2pMessage.Topic, hash[:8])

	case protobuf.ResponseType_MessageTraceGossipSub:
		handleGossipSubTrace(resp.GetData())
	case protobuf.ResponseType_MessageTraceOptimumP2P:
		handleOptimumP2PTrace(resp.GetData())
	case protobuf.ResponseType_Unknown:
	default:
		log.Println("Unknown response command:", resp.GetCommand())
	}
}

func headHex(b []byte, n int) string {
	if len(b) > n {
		b = b[:n]
	}
	return hex.EncodeToString(b)
}

func handleGossipSubTrace(data []byte) {
	evt := &pubsubpb.TraceEvent{}
	if err := proto.Unmarshal(data, evt); err != nil {
		fmt.Printf("[TRACE] GossipSub decode error: %v raw=%dB head=%s\n",
			err, len(data), headHex(data, 64))
		return
	}

	ts := time.Unix(0, evt.GetTimestamp()).Format(time.RFC3339Nano)
	// print type
	fmt.Printf("[TRACE] GossipSub type=%s ts=%s size=%dB\n", evt.GetType().String(), ts, len(data))
	jb, _ := json.Marshal(evt)
	fmt.Printf("[TRACE] GossipSub JSON (%dB): %s\n", len(jb), string(jb))
}

func handleOptimumP2PTrace(data []byte) {
	evt := &optsub.TraceEvent{}
	if err := proto.Unmarshal(data, evt); err != nil {
		fmt.Printf("[TRACE] OptimumP2P decode error: %v\n", err)
		return
	}

	// human-readable timestamp
	ts := time.Unix(0, evt.GetTimestamp()).Format(time.RFC3339Nano)

	// print type
	typeStr := optsub.TraceEvent_Type_name[int32(evt.GetType())]
	fmt.Printf("[TRACE] OptimumP2P type=%s ts=%s size=%dB\n", typeStr, ts, len(data))

	// if shard-related
	switch evt.GetType() {
	case optsub.TraceEvent_NEW_SHARD:
		fmt.Printf("  NEW_SHARD id=%x coeff=%x\n", evt.GetNewShard().GetMessageID(), evt.GetNewShard().GetCoefficients())
	case optsub.TraceEvent_DUPLICATE_SHARD:
		fmt.Printf("  DUPLICATE_SHARD id=%x\n", evt.GetDuplicateShard().GetMessageID())
	case optsub.TraceEvent_UNHELPFUL_SHARD:
		fmt.Printf("  UNHELPFUL_SHARD id=%x\n", evt.GetUnhelpfulShard().GetMessageID())
	case optsub.TraceEvent_UNNECESSARY_SHARD:
		fmt.Printf("  UNNECESSARY_SHARD id=%x\n", evt.GetUnnecessaryShard().GetMessageID())
	}

	jb, _ := json.Marshal(evt)
	fmt.Printf("[TRACE] OptimumP2P JSON (%dB): %s\n", len(jb), string(jb))
}
