package main

import (
	"bufio"
	"context"
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
	topic  = flag.String("topic", "", "topic name")
	ipfile = flag.String("ipfile", "", "file with a list of IP addresses")
	startIdx = flag.Int("start-index", 0, "default 0" )
	endIdx = flag.Int("end-index", 10000, "default 0" )
)

func main() {
	flag.Parse()
	if *topic == "" {
		log.Fatalf("−topic is required")
	}

	_ips, err := readIPsFromFile(*ipfile)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
        fmt.Printf("numip %d  index %d\n",len(_ips), *endIdx)
        *endIdx = min(len(_ips), *endIdx)
	ips := _ips[*startIdx:*endIdx]
	fmt.Printf("Found %d IPs: %v\n", len(ips), ips)


	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\nShutting down gracefully…")
		cancel()
	}()

	// Launch goroutines with synchronization
	var wg sync.WaitGroup
	for _, ip := range ips {
		wg.Add(1)
		go func(ip string) {
			defer wg.Done()
			data := ip
			receiveMessages(ctx, ip, data)
		}(ip)
	}

	wg.Wait()
}

func receiveMessages(ctx context.Context, ip string, message string) error {
	// connect with simple gRPC settings
	//fmt.Println("Starting ", ip)
	select {
	case <-ctx.Done():
		log.Printf("[%s] context canceled, stopping", ip)
		return ctx.Err()
	default:
	}

	conn, err := grpc.NewClient(ip,
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

	stream, err := client.ListenCommands(ctx)

	if err != nil {
		log.Fatalf("ListenCommands: %v", err)
	}

	println(fmt.Sprintf("Connected to node at: %s…", ip))
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
			return nil
		case resp, ok := <-msgChan:
			if !ok {
				log.Printf("Stream closed. Total messages received: %d", atomic.LoadInt32(&receivedCount))
				return nil
			}
			go func(resp *protobuf.Response) {
				handleResponse(ip, resp, &receivedCount)
			}(resp)
		}
	}

	return nil
}

func readIPsFromFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var ips []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		ips = append(ips, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return ips, nil
}

func handleResponse(ip string, resp *protobuf.Response, counter *int32) {
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

		//fmt.Printf("Recv message: [%d] [%d %d] %s\n\n",n,  currentTime, messageSize, string(p2pMessage.Message)[0:100])
		fmt.Printf("Recv message: [%s] [%d] [%d %d] %s\n\n", ip, n, currentTime, messageSize, string(p2pMessage.Message))
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
