package shared

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync/atomic"
	"time"

	protobuf "p2p_client/grpc"
	optsub "p2p_client/grpc/mump2p_trace"

	"github.com/gogo/protobuf/proto"
	pubsubpb "github.com/libp2p/go-libp2p-pubsub/pb"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/mr-tron/base58"
)

func ReadIPsFromFile(filename string) ([]string, error) {
	if strings.TrimSpace(filename) == "" {
		return nil, fmt.Errorf("-ipfile is required")
	}

	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var ips []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
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

func HeadHex(b []byte, n int) string {
	if len(b) > n {
		b = b[:n]
	}
	return hex.EncodeToString(b)
}

func HandleResponse(resp *protobuf.Response, counter *int32) {
	switch resp.GetCommand() {
	case protobuf.ResponseType_Message:
		var p2pMessage P2PMessage
		if err := json.Unmarshal(resp.GetData(), &p2pMessage); err != nil {
			log.Printf("Error unmarshalling message: %v", err)
			return
		}
		n := atomic.AddInt32(counter, 1)
		messageSize := len(p2pMessage.Message)
		currentTime := time.Now().UnixNano()
		fmt.Printf("Recv message: [%d] [%d %d] %s\n\n", n, currentTime, messageSize, string(p2pMessage.Message))
	case protobuf.ResponseType_MessageTraceGossipSub:
		log.Printf("GossipSub trace received but handler not implemented")
	case protobuf.ResponseType_MessageTraceMumP2P:
		log.Printf("MumP2P trace received but handler not implemented")
	case protobuf.ResponseType_Unknown:
	default:
		log.Println("Unknown response command:", resp.GetCommand())
	}
}

func HandleResponseWithTracking(ip string, resp *protobuf.Response, counter *int32,
	writeData bool, dataCh chan<- string, writeTrace bool, traceCh chan<- string) {

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
		if len(parts) > 0 && writeData {
			publisher := parts[0]
			dataToSend := fmt.Sprintf("%s\t%s\t%d\t%s", ip, publisher, len(p2pMessage.Message), hexHashString)
			dataCh <- dataToSend
		}

	case protobuf.ResponseType_MessageTraceMumP2P:
		HandleOptimumP2PTrace(resp.GetData(), writeTrace, traceCh)
	case protobuf.ResponseType_MessageTraceGossipSub:
		HandleGossipSubTrace(resp.GetData(), writeTrace, traceCh)
	default:
		log.Println("Unknown response command:", resp.GetCommand())
	}
}

func HandleGossipSubTrace(data []byte, writeTrace bool, traceCh chan<- string) {
	evt := &pubsubpb.TraceEvent{}
	if err := proto.Unmarshal(data, evt); err != nil {
		fmt.Printf("[TRACE] GossipSub decode error: %v raw=%dB head=%s\n",
			err, len(data), HeadHex(data, 64))
		return
	}

	typeStr := optsub.TraceEvent_Type_name[int32(evt.GetType())]
	var peerID peer.ID
	if evt.PeerID != nil {
		rawBytes := []byte(evt.PeerID)
		peerID = peer.ID(rawBytes)
	}

	recvID := ""
	if evt.DeliverMessage != nil && evt.DeliverMessage.ReceivedFrom != nil {
		rawBytes := []byte(evt.DeliverMessage.ReceivedFrom)
		recvID = base58.Encode(rawBytes)
	}

	msgID := ""
	topic := ""
	if evt.DeliverMessage != nil {
		rawBytes := []byte(evt.DeliverMessage.MessageID)
		msgID = base58.Encode(rawBytes)
		topic = string(*evt.DeliverMessage.Topic)
	}
	if evt.PublishMessage != nil {
		rawBytes := []byte(evt.PublishMessage.MessageID)
		msgID = base58.Encode(rawBytes)
		topic = string(*evt.PublishMessage.Topic)
	}

	timestamp := int64(0)
	if evt.Timestamp != nil {
		timestamp = *evt.Timestamp
	}

	if writeTrace {
		dataToSend := fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%d", typeStr, peerID, recvID, msgID, topic, timestamp)
		traceCh <- dataToSend
	} else {
		fmt.Printf("%s\t%s\t%s\t%s\t%s\t%d\n", typeStr, peerID, recvID, msgID, topic, timestamp)
	}
}

func HandleOptimumP2PTrace(data []byte, writeTrace bool, traceCh chan<- string) {
	evt := &optsub.TraceEvent{}
	if err := proto.Unmarshal(data, evt); err != nil {
		fmt.Printf("[TRACE] mump2p decode error: %v\n", err)
		return
	}

	typeStr := optsub.TraceEvent_Type_name[int32(evt.GetType())]

	var peerID peer.ID
	if evt.PeerID != nil {
		rawBytes := []byte(evt.PeerID)
		peerID = peer.ID(rawBytes)
	}

	recvID := ""
	if evt.DeliverMessage != nil && evt.DeliverMessage.ReceivedFrom != nil {
		rawBytes := []byte(evt.DeliverMessage.ReceivedFrom)
		recvID = base58.Encode(rawBytes)
	}
	if evt.NewShard != nil && evt.NewShard.ReceivedFrom != nil {
		rawBytes := []byte(evt.NewShard.ReceivedFrom)
		recvID = base58.Encode(rawBytes)
	}

	msgID := ""
	topic := ""
	if evt.DeliverMessage != nil {
		rawBytes := []byte(evt.DeliverMessage.MessageID)
		msgID = base58.Encode(rawBytes)
		topic = string(*evt.DeliverMessage.Topic)
	}
	if evt.PublishMessage != nil {
		rawBytes := []byte(evt.PublishMessage.MessageID)
		msgID = base58.Encode(rawBytes)
		topic = string(*evt.PublishMessage.Topic)
	}
	if evt.NewShard != nil {
		rawBytes := []byte(evt.NewShard.MessageID)
		msgID = base58.Encode(rawBytes)
	}

	timestamp := int64(0)
	if evt.Timestamp != nil {
		timestamp = *evt.Timestamp
	}

	if writeTrace {
		dataToSend := fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%d", typeStr, peerID, recvID, msgID, topic, timestamp)
		traceCh <- dataToSend
	} else {
		fmt.Printf("%s\t%s\t%s\t%s\t%s\t%d\n", typeStr, peerID, recvID, msgID, topic, timestamp)
	}
}

func WriteToFile(ctx context.Context, dataCh <-chan string, done chan<- bool, filename string, header string) {
	file, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	defer close(done)

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	if header != "" {
		_, err := writer.WriteString(header + "\n")
		if err != nil {
			log.Printf("Write error: %v", err)
		}
	}

	ctxDone := ctx.Done()
	for {
		select {
		case <-ctxDone:
			// Continue draining channel so producers don't block on shutdown.
			ctxDone = nil
		case data, ok := <-dataCh:
			if !ok {
				fmt.Println("All data flushed to disk")
				return
			}

			_, err := writer.WriteString(data + "\n")
			writer.Flush()
			if err != nil {
				log.Printf("Write error: %v", err)
			}
		}
	}
}
