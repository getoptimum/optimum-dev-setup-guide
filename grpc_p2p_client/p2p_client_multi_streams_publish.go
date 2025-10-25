package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
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
	topic = flag.String("topic", "", "topic name")

	// optional: number of messages to publish (for stress testing or batch sending)
	count    = flag.Int("count", 1, "number of messages to publish")
	dataSize = flag.Int("datasize", 100, "size of random of messages to publish")
	// optional: sleep duration between publishes
	sleep    = flag.Duration("sleep", 50*time.Millisecond, "optional delay between publishes (e.g., 1s, 500ms)")
	ipfile   = flag.String("ipfile", "", "file with a list of IP addresses")
	startIdx = flag.Int("start-index", 0, "beginning index is 0: default 0")
	endIdx   = flag.Int("end-index", 10000, "index-1")
	output   = flag.String("output", "", "file to write the outgoing data hashes")
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
	done := make(chan bool)

	var wg sync.WaitGroup
	// Start writing the has of the published data
	if *output != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			go writeHashToFile(dataCh, done, *output)
		}()
	}

	// Launch goroutines with synchronization
	for _, ip := range ips {
		wg.Add(1)
		go func(ip string) {
			defer wg.Done()
			datasize := *dataSize
			sendMessages(ctx, ip, datasize, *output != "", dataCh)
		}(ip)
	}
	wg.Wait()
	close(dataCh)
	<-done

}

func sendMessages(ctx context.Context, ip string, datasize int, write bool, dataCh chan<- string) error {
	// connect with simple gRPC settings
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
	println(fmt.Sprintf("Connected to node at: %s…", ip))

	client := protobuf.NewCommandStreamClient(conn)

	stream, err := client.ListenCommands(ctx)

	if err != nil {
		log.Fatalf("ListenCommands: %v", err)
	}

	for i := 0; i < *count; i++ {
		start := time.Now()
		var data []byte
		//currentTime := time.Now().UnixNano()
		randomBytes := make([]byte, datasize)
		if _, err := rand.Read(randomBytes); err != nil {
			log.Fatalf("failed to generate random bytes: %v", err)
		}

		randomSuffix := hex.EncodeToString(randomBytes)
		data = []byte(fmt.Sprintf("%s-%s", ip, randomSuffix))
		pubReq := &protobuf.Request{
			Command: int32(CommandPublishData),
			Topic:   *topic,
			Data:    data,
		}

		if err := stream.Send(pubReq); err != nil {
			log.Fatalf("send publish: %v", err)
		}

		elapsed := time.Since(start)

		hash := sha256.Sum256(data)
		hexHashString := hex.EncodeToString(hash[:])
		var dataToSend string
		if write == true {
			dataToSend = fmt.Sprintf("%s\t%d\t%s", ip, len(data), hexHashString)
			dataCh <- dataToSend
		}
		fmt.Printf("Published %s to %q (took %v)\n", dataToSend, *topic, elapsed)

		if *sleep > 0 {
			time.Sleep(*sleep)
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

		//fmt.Printf("Recv message: [%d] [%d %d] %s\n\n",n,  currentTime, messageSize, string(p2pMessage.Message)[0:100])
		fmt.Printf("Recv message: [%d] [%d %d] %s\n\n", n, currentTime, messageSize, string(p2pMessage.Message))
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

func writeHashToFile(dataCh <-chan string, done chan<- bool, filename string) {
	file, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	// Process until channel is closed
	for data := range dataCh {
		_, err := writer.WriteString(data + "\n")
		if err != nil {
			log.Printf("Write error: %v", err)
		}
	}
	done <- true
	fmt.Println("All data flushed to disk")

}
