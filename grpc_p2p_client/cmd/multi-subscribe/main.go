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
		log.Fatalf("−topic is required")
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

	var wg sync.WaitGroup
	if *outputData != "" {
		dataDone = make(chan bool)
		go func() {
			header := "receiver\tsender\tsize\tsha256(msg)"
			go shared.WriteToFile(ctx, dataCh, dataDone, *outputData, header)
		}()
	}

	if *outputTrace != "" {
		traceDone = make(chan bool)
		go func() {
			header := ""
			go shared.WriteToFile(ctx, traceCh, traceDone, *outputTrace, header)
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
	if dataDone != nil {
		<-dataDone
	}
	if traceDone != nil {
		<-traceDone
	}
}

func receiveMessages(ctx context.Context, ip string, writeData bool, dataCh chan<- string,
	writeTrace bool, traceCh chan<- string) error {

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
	msgChan := make(chan *protobuf.Response, 10000)

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
				shared.HandleResponseWithTracking(ip, resp, &receivedCount, writeData, dataCh, writeTrace, traceCh)
			}(resp)
		}
	}
}
