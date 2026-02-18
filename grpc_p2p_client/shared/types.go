package shared

// P2PMessage represents a message structure used in P2P communication
type P2PMessage struct {
	MessageID    string
	Topic        string
	Message      []byte
	SourceNodeID string
}

type Message struct {
	Topic string `json:"topic"`
	Msg   []byte `json:"msg"`
}

type MessageBatch struct {
	Messages []Message `json:"messages"`
}

// Command represents possible operations that sidecar may perform with p2p node
type Command int32

const (
	CommandUnknown Command = iota
	CommandPublishData
	CommandSubscribeToTopic
	CommandUnSubscribeToTopic
	CommandSubscribeToTopics
	CommandPublishRandomData
	CommandPublishBatch
)
