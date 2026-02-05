package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"

	protobuf "p2p_client/grpc"
	"p2p_client/shared"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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
		log.Fatal("-topic is required")
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
	traceCh := make(chan string, 100)
	var dataDone chan bool
	var traceDone chan bool
	errCh := make(chan error, len(ips))

	var wg sync.WaitGroup
	if *outputData != "" {
		dataDone = make(chan bool)
		header := "receiver\tsender\tsize\tsha256(msg)"
		go shared.WriteToFile(ctx, dataCh, dataDone, *outputData, header)
	}

	if *outputTrace != "" {
		traceDone = make(chan bool)
		header := ""
		go shared.WriteToFile(ctx, traceCh, traceDone, *outputTrace, header)
	}

	for _, ip := range ips {
		wg.Add(1)
		go func(ip string) {
			defer wg.Done()
			if err := receiveMessages(ctx, ip, *outputData != "", dataCh, *outputTrace != "", traceCh); err != nil {
				errCh <- err
				cancel()
			}
		}(ip)
	}

	wg.Wait()
	close(errCh)
	close(dataCh)
	close(traceCh)
	if dataDone != nil {
		<-dataDone
	}
	if traceDone != nil {
		<-traceDone
	}

	hasErrors := false
	for err := range errCh {
		hasErrors = true
		log.Printf("subscribe worker error: %v", err)
	}
	if hasErrors {
		os.Exit(1)
	}
}

func receiveMessages(ctx context.Context, ip string, writeData bool, dataCh chan<- string,
	writeTrace bool, traceCh chan<- string) error {

	select {
	case <-ctx.Done():
		log.Printf("[%s] context canceled, stopping", ip)
		return nil
	default:
	}

	conn, err := grpc.NewClient(ip,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(math.MaxInt),
			grpc.MaxCallSendMsgSize(math.MaxInt),
		),
	)

	fmt.Printf("IP -  %v\n", ip)
	if err != nil {
		log.Printf("[%s] failed to connect to node: %v", ip, err)
		return fmt.Errorf("failed to connect to node %s: %w", ip, err)
	}
	defer conn.Close()

	client := protobuf.NewCommandStreamClient(conn)
	stream, err := client.ListenCommands(ctx)
	if err != nil {
		log.Printf("[%s] ListenCommands failed: %v", ip, err)
		return fmt.Errorf("ListenCommands failed for %s: %w", ip, err)
	}

	println(fmt.Sprintf("Connected to node at: %s…", ip))
	println(fmt.Sprintf("Trying to subscribe to topic %s…", *topic))
	subReq := &protobuf.Request{
		Command: int32(shared.CommandSubscribeToTopic),
		Topic:   *topic,
	}
	if err := stream.Send(subReq); err != nil {
		log.Printf("[%s] send subscribe failed: %v", ip, err)
		return fmt.Errorf("send subscribe failed for %s: %w", ip, err)
	}
	fmt.Printf("Subscribed to topic %q, waiting for messages…\n", *topic)

	var receivedCount int32
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			log.Printf("[%s] stream closed. Total messages received: %d", ip, atomic.LoadInt32(&receivedCount))
			return nil
		}
		if err != nil {
			if ctx.Err() != nil {
				log.Printf("[%s] context canceled. Total messages received: %d", ip, atomic.LoadInt32(&receivedCount))
				return nil
			}
			return fmt.Errorf("[%s] recv error: %w", ip, err)
		}

		shared.HandleResponseWithTracking(ip, resp, &receivedCount, writeData, dataCh, writeTrace, traceCh)
	}
}
