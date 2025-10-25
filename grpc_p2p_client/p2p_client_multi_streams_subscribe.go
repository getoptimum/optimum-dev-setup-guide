package main

import (
	"bufio"
	"context"
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
	"strings"
	"sync"
	"sync/atomic"
	"syscall"

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
	topic       = flag.String("topic", "", "topic name")
	ipfile      = flag.String("ipfile", "", "file with a list of IP addresses")
	startIdx    = flag.Int("start-index", 0, "beginning index is 0: default 0")
	endIdx      = flag.Int("end-index", 10000, "index-1")
	outputTrace = flag.String("output-trace", "", "file to write the outgoing data hashes")
	outputData  = flag.String("output-data", "", "file to write the outgoing data hashes")
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
	fmt.Printf("numip %d  index %d\n", len(_ips), *endIdx)
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

	// Buffered channel to prevent blocking
	dataCh := make(chan string, 100)
	traceCh := make(chan string, 100)
	dataDone := make(chan bool)
	traceDone := make(chan bool)

	// Launch goroutines with synchronization
	var wg sync.WaitGroup
	if *outputData != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			go writeToFile(ctx, dataCh, dataDone, *outputData)
		}()
	}

	if *outputTrace != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			go writeToFile(ctx, traceCh, traceDone, *outputTrace)
		}()
	}

	for _, ip := range ips {
		wg.Add(1)
		go func(ip string) {
			defer wg.Done()
			receiveMessages(ctx, ip, *outputData != "", dataCh, *outputTrace != "", traceCh)
		}(ip)
	}

	wg.Wait()
	close(dataCh)
	close(traceCh)
	<-dataDone
	<-traceDone
}

func receiveMessages(ctx context.Context, ip string, writeData bool, dataCh chan<- string,
	writeTrace bool, traceCh chan<- string) error {
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
				handleResponse(ip, resp, &receivedCount, writeData, dataCh, writeTrace, traceCh)
			}(resp)
		}
	}

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

func handleResponse(ip string, resp *protobuf.Response, counter *int32,
	writedata bool, dataCh chan<- string, writetrace bool, traceCh chan<- string) {

	switch resp.GetCommand() {
	case protobuf.ResponseType_Message:
		var p2pMessage P2PMessage
		if err := json.Unmarshal(resp.GetData(), &p2pMessage); err != nil {
			log.Printf("Error unmarshalling message: %v", err)
			return
		}
		_ = atomic.AddInt32(counter, 1)

		hash := sha256.Sum256(p2pMessage.Message)
		hexHashString := hex.EncodeToString(hash[:])

		parts := strings.Split(string(p2pMessage.Message), "-")
		if len(parts) > 0 {
			publisher := parts[0]
			var dataToSend string
			if writedata == true {
				dataToSend = fmt.Sprintf("%s\t%s\t%d\t%s", ip, publisher, len(p2pMessage.Message), hexHashString)
				dataCh <- dataToSend
			}
		}

		//fmt.Printf("Recv message: %s %d %s\n", ip,  messageSize, string(p2pMessage.Message))
	case protobuf.ResponseType_MessageTraceOptimumP2P:
		handleOptimumP2PTrace(resp.GetData(), writetrace, traceCh)
	case protobuf.ResponseType_MessageTraceGossipSub:
		handleGossipSubTrace(resp.GetData(), writetrace, traceCh)
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

func handleGossipSubTrace(data []byte, writetrace bool, traceCh chan<- string) {
	evt := &pubsubpb.TraceEvent{}
	if err := proto.Unmarshal(data, evt); err != nil {
		fmt.Printf("[TRACE] GossipSub decode error: %v raw=%dB head=%s\n",
			err, len(data), headHex(data, 64))
		return
	}
	typeStr := optsub.TraceEvent_Type_name[int32(evt.GetType())]
	//fmt.Printf("[TRACE] GossipSub type=%s ts=%s size=%dB\n", evt.GetType().String(), ts, len(data))
	//fmt.Printf("[TRACE] GossipSub JSON (%dB): %s\n", len(jb), string(jb))

	jb, _ := json.Marshal(evt)
	//fmt.Printf("[TRACE] GossipSub JSON message_type=%s, (%dB): %s\n", typeStr, len(jb), string(jb))
	if writetrace {
		dataToSend := fmt.Sprintf("[TRACE] GossipSub JSON message_type=%s, (%dB): %s", typeStr, len(jb), string(jb))
		traceCh <- dataToSend
	} else {
		fmt.Printf("[TRACE] GossipSub JSON message_type=%s, (%dB): %s\n", typeStr, len(jb), string(jb))
	}

}

func handleOptimumP2PTrace(data []byte, writetrace bool, traceCh chan<- string) {
	evt := &optsub.TraceEvent{}
	if err := proto.Unmarshal(data, evt); err != nil {
		fmt.Printf("[TRACE] OptimumP2P decode error: %v\n", err)
		return
	}

	// print type
	typeStr := optsub.TraceEvent_Type_name[int32(evt.GetType())]
	//fmt.Printf("[TRACE] OptimumP2P type=%s ts=%s size=%dB\n", typeStr, ts, len(data))
	//fmt.Printf("[TRACE] OptimumP2P type=%s msg_id=%x time=%d, recvr_id=%s, size=%dB\n",
	//		typeStr, evt.GetDuplicateShard().GetMessageID(), time.Unix(0, evt.GetTimestamp()), evt.GetPeerID(), len(data))

	// if shard-related
	/*
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
	*/

	jb, _ := json.Marshal(evt)
	if writetrace {
		dataToSend := fmt.Sprintf("[TRACE] OptimumP2P JSON message_type=%s, (%dB): %s", typeStr, len(jb), string(jb))
		traceCh <- dataToSend
	} else {
		fmt.Printf("[TRACE] OptimumP2P JSON message_type=%s, (%dB): %s\n", typeStr, len(jb), string(jb))

	}

	/*
		     message_type  <- systems information
		     message_id   <- application layer
	     	     time_stamp  <- event occuring the event  publish, new shard, duplicate shard
		     receiver_id
		     sender_id

	*/

}

func writeToFile(ctx context.Context, dataCh <-chan string, done chan<- bool, filename string) {
	file, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	// Process until channel is closed
	for data := range dataCh {
		select {
		case <-ctx.Done():
			return
		default:

		}
		_, err := writer.WriteString(data + "\n")
		writer.Flush()
		if err != nil {
			log.Printf("Write error: %v", err)
		}
	}
	done <- true
	fmt.Println("All data flushed to disk")
}
