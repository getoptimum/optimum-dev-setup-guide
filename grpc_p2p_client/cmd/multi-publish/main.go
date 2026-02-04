package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"math"
	mathrand "math/rand"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	protobuf "p2p_client/grpc"
	"p2p_client/shared"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	topic    = flag.String("topic", "", "topic name")
	count    = flag.Int("count", 1, "number of messages to publish")
	poisson  = flag.Bool("poisson", false, "Enable Poisson arrival")
	dataSize = flag.Int("datasize", 100, "size of random of messages to publish")
	sleep    = flag.Duration("sleep", 50*time.Millisecond, "optional delay between publishes (e.g., 1s, 500ms)")
	ipfile   = flag.String("ipfile", "", "file with a list of IP addresses")
	startIdx = flag.Int("start-index", 0, "beginning index is 0: default 0")
	endIdx   = flag.Int("end-index", 10000, "index-1")
	output   = flag.String("output", "", "file to write the outgoing data hashes")
)

func main() {
	flag.Parse()
	if *topic == "" {
		log.Fatal("-topic is required")
	}
	if *count < 1 {
		log.Fatal("-count must be >= 1")
	}
	if *dataSize < 1 {
		log.Fatal("-datasize must be >= 1")
	}

	_ips, err := shared.ReadIPsFromFile(*ipfile)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("numip %d  index %d\n", len(_ips), *endIdx)
	*endIdx = min(len(_ips), *endIdx)
	if *startIdx < 0 || *startIdx >= *endIdx || *startIdx >= len(_ips) {
		log.Fatalf("invalid index range: start-index=%d end-index=%d (num IPs=%d)", *startIdx, *endIdx, len(_ips))
	}

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

	dataCh := make(chan string, 100)
	randomByteLen := max(1, *dataSize/2)
	var done chan bool
	var wg sync.WaitGroup
	errCh := make(chan error, len(ips))

	if *output != "" {
		done = make(chan bool)
		header := "sender\tsize\tsha256(msg)"
		go shared.WriteToFile(ctx, dataCh, done, *output, header)
	}

	for _, ip := range ips {
		wg.Add(1)
		go func(ip string) {
			defer wg.Done()
			if err := sendMessages(ctx, ip, randomByteLen, *output != "", dataCh); err != nil {
				errCh <- err
				cancel()
			}
		}(ip)
	}
	wg.Wait()
	close(errCh)
	close(dataCh)
	if done != nil {
		<-done
	}

	hasErrors := false
	for err := range errCh {
		hasErrors = true
		log.Printf("publish worker error: %v", err)
	}
	if hasErrors {
		os.Exit(1)
	}
}

func sendMessages(ctx context.Context, ip string, datasize int, write bool, dataCh chan<- string) error {
	// Create connection once and reuse for all messages
	conn, err := grpc.NewClient(ip,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(math.MaxInt),
			grpc.MaxCallSendMsgSize(math.MaxInt),
		),
	)
	if err != nil {
		return fmt.Errorf("[%s] failed to connect to node: %w", ip, err)
	}
	defer conn.Close()

	client := protobuf.NewCommandStreamClient(conn)
	stream, err := client.ListenCommands(ctx)
	if err != nil {
		return fmt.Errorf("[%s] ListenCommands failed: %w", ip, err)
	}

	println(fmt.Sprintf("Connected to node at: %s…", ip))

	for i := 0; i < *count; i++ {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		start := time.Now()
		randomBytes := make([]byte, datasize)
		if _, err := rand.Read(randomBytes); err != nil {
			return fmt.Errorf("[%s] failed to generate random bytes: %w", ip, err)
		}

		randomSuffix := hex.EncodeToString(randomBytes)
		data := []byte(fmt.Sprintf("%s-%s", ip, randomSuffix))
		pubReq := &protobuf.Request{
			Command: int32(shared.CommandPublishData),
			Topic:   *topic,
			Data:    data,
		}

		if err := stream.Send(pubReq); err != nil {
			return fmt.Errorf("[%s] send publish: %w", ip, err)
		}

		elapsed := time.Since(start)
		hash := sha256.Sum256(data)
		hexHashString := hex.EncodeToString(hash[:])
		if write {
			dataToSend := fmt.Sprintf("%s\t%d\t%s", ip, len(data), hexHashString)
			dataCh <- dataToSend
		}
		fmt.Printf("[%s] published %d bytes to %q (took %v)\n", ip, len(data), *topic, elapsed)

		if *poisson {
			lambda := 1.0 / (*sleep).Seconds()
			interval := mathrand.ExpFloat64() / lambda
			waitTime := time.Duration(interval * float64(time.Second))
			time.Sleep(waitTime)
		} else {
			time.Sleep(*sleep)
		}
	}

	return nil
}
