package main

import (
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
	"strconv"
	"strings"
	"syscall"
	"time"

	protobuf "p2p_client/grpc"
	"p2p_client/shared"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	addr        = flag.String("addr", "localhost:33212", "sidecar gRPC address")
	topics      = flag.String("topics", "", "topic names")
	messageSize = flag.String("msg", "", "size per message (for publish)")
	output      = flag.String("output", "", "file to write the outgoing data hashes")
	sleep       = flag.Duration("sleep", 12*time.Second, "delay between batches (e.g. 12s)")
	numBatches  = flag.Int("num_batches", 1, "number of batches to publish")
)

func validateFlags() {
	if *topics == "" {
		log.Fatal("-topics is required")
	}
	if *messageSize == "" {
		log.Fatal("-msg is required")
	}
	if *addr == "" {
		log.Fatal("-addr is required")
	}
	if *sleep < 0 {
		log.Fatal("-sleep must be >= 0")
	}
	if *numBatches < 1 {
		log.Fatal("-num_batches must be >= 1")
	}
}

func main() {
	flag.Parse()
	validateFlags()
	topics := parseTopics(*topics)
	if len(topics) == 0 {
		log.Fatal("no topics provided")
	}
	msgSize, err := parseMessageSize(*messageSize)
	if err != nil {
		log.Fatalf("invalid message size: %v", err)
	}

	fmt.Printf("Connecting to node at: %s…\n", *addr)
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
	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("error closing connection: %v", err)
		}
	}()

	client := protobuf.NewCommandStreamClient(conn)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\nshutting down…")
		cancel()
	}()
	stream, err := client.ListenCommands(ctx)
	if err != nil {
		log.Fatalf("ListenCommands: %v", err)
	}

	var done chan bool
	dataCh := make(chan string, 100)
	if *output != "" {
		done = make(chan bool)
		header := "sender\tsize\tsha256(msg)"
		go shared.WriteToFile(ctx, dataCh, done, *output, header)
	}
	for i := 0; i < *numBatches; i++ {
		if err := batchPublish(ctx, stream, topics, msgSize, *output != "", dataCh); err != nil {
			fmt.Printf("batch publish error: %v", err)
			cancel()
		}
		time.Sleep(*sleep)
	}
	if err := stream.CloseSend(); err != nil {
		fmt.Printf("failed to close send side of stream: %v", err)
	}
	fmt.Printf("\nBatch publish completed: %d batches published (%d messages total, %d bytes total)\n", *numBatches, *numBatches*len(topics), *numBatches*len(topics)*msgSize)

	close(dataCh)
	if done != nil {
		<-done
	}
}

func parseTopics(topicsStr string) []string {
	if topicsStr == "" {
		return nil
	}
	topics := strings.Split(topicsStr, ",")
	result := make([]string, 0, len(topics))
	for _, t := range topics {
		t = strings.TrimSpace(t)
		if t != "" {
			result = append(result, t)
		}
	}
	return result
}

func parseMessageSize(msgSizeStr string) (int, error) {
	size, err := strconv.Atoi(msgSizeStr)
	if err != nil {
		return 0, err
	}
	return size, nil
}

func batchPublish(ctx context.Context, stream protobuf.CommandStream_ListenCommandsClient, topics []string, messageSize int, write bool, dataCh chan<- string) error {
	fmt.Printf("Batch publishing to %d topics: %v\n", len(topics), topics)
	select {
	case <-ctx.Done():
		fmt.Println("Context canceled, stopping batch publish")
		return fmt.Errorf("context canceled")
	default:
	}

	start := time.Now()
	messages := make([]shared.Message, 0, len(topics))
	for _, topic := range topics {
		randomBytes := make([]byte, messageSize)
		if _, err := rand.Read(randomBytes); err != nil {
			return fmt.Errorf("failed to generate random bytes: %v", err)
		}
		randomSuffix := hex.EncodeToString(randomBytes[:min(len(randomBytes), 8)])
		currentTime := time.Now().UnixNano()
		data := []byte(fmt.Sprintf("[%d %d] topic:%s - %s", currentTime, messageSize, topic, randomSuffix))

		if len(data) < messageSize {
			padding := make([]byte, messageSize-len(data))
			_, err := rand.Read(padding)
			if err != nil {
				return fmt.Errorf("failed to generate random bytes: %v", err)
			}
			data = append(data, padding...)
		} else if len(data) > messageSize {
			data = data[:messageSize]
		}

		messages = append(messages, shared.Message{
			Topic: topic,
			Msg:   data,
		})
	}

	batch := shared.MessageBatch{
		Messages: messages,
	}
	batchData, err := json.Marshal(batch)
	if err != nil {
		return fmt.Errorf("failed to marshal batch: %v", err)
	}
	batchReq := &protobuf.Request{
		Command: int32(shared.CommandPublishBatch),
		Data:    batchData,
	}
	if err := stream.Send(batchReq); err != nil {
		return fmt.Errorf("send batch publish: %v", err)
	}

	elapsed := time.Since(start)
	hash := sha256.Sum256(batchData)
	hexHashString := hex.EncodeToString(hash[:])
	if write {
		dataToSend := fmt.Sprintf("%d\t%s", len(batchData), hexHashString)
		dataCh <- dataToSend
	}
	fmt.Printf("Published batch to %d topics (%d bytes, took %v)\n", len(topics), len(batchData), elapsed)

	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
