# Metrics Reference

## Core/Process

| Metric Name                                           | Type  | Labels | How It’s Calculated                          | Goal                               |
| ----------------------------------------------------- | ----- | ------ | -------------------------------------------- | ---------------------------------- |
| `optimum_<subsystem>_system_cpu_cores`                | Gauge | none   | Set once at init to `runtime.NumCPU()`       | Machine capacity context           |
| Go / process collectors                               | —     | —      | From `collectors.NewGoCollector()` / Process | Runtime health & resource usage    |
| `optimum_mump2p_delivered_messages_delay_seconds`     | Hist  | none   | `ObserveMumP2PMessageDelay(seconds)`         | End-to-end delivery delay (mumP2P) |
| `optimum_gossip-p2p_delivered_messages_delay_seconds` | Hist  | none   | `ObserveGossipMessageDelay(seconds)`         | End-to-end delivery delay (GS)     |

## Geo-location

| Metric Name                            | Type  | Labels (`country_iso`,`latitude`,`longitude`) | How It’s Calculated                     | Goal                       |
| -------------------------------------- | ----- | --------------------------------------------- | --------------------------------------- | -------------------------- |
| `optimum_det_coordinates_geo_location` | Gauge | yes                                           | Periodic fetch from geolocation service | Geo filtering & dashboards |

## Bandwidth (libp2p stream I/O)

| Metric Name                             | Type       | Labels (`direction`) / (`protocol`,`direction`) | How It’s Calculated                             | Goal                              |
| --------------------------------------- | ---------- | ----------------------------------------------- | ----------------------------------------------- | --------------------------------- |
| `optimum_bandwidth_total_bytes`         | CounterVec | `direction`                                     | `LogSentMessageStream` / `LogRecvMessageStream` | Total bytes in/out                |
| `optimum_bandwidth_messages_total`      | CounterVec | `protocol`,`direction`                          | Same as above (increments by 1 per message)     | Msg count by protocol & direction |
| `optimum_bandwidth_traffic_bytes_total` | CounterVec | `protocol`,`direction`                          | Add `size` per call                             | Bytes by protocol & direction     |

## Connections & Streams

| Metric Name                            | Type     | Labels      | How It’s Calculated                       | Goal                           |
| -------------------------------------- | -------- | ----------- | ----------------------------------------- | ------------------------------ |
| `optimum_conn_total_connections`       | GaugeVec | `direction` | `Connected`++ / `Disconnected`--          | Live libp2p conns by direction |
| `optimum_conn_streams_current`         | GaugeVec | `protocol`  | `OpenedStream`++ / `ClosedStream`--       | Live streams by protocol       |
| `optimum_conn_stream_duration_seconds` | HistVec  | `protocol`  | On `ClosedStream`: observe `now - opened` | Stream lifetime distribution   |

## GossipSub

| Metric Name                                   | Type       | Labels     | How It’s Calculated                                 | Goal                            |
| --------------------------------------------- | ---------- | ---------- | --------------------------------------------------- | ------------------------------- |
| `optimum_gossip-p2p_total_peers`              | Gauge      | none       | `AddPeer`++ / `RemovePeer`--                        | Total GS peers                  |
| `optimum_gossip-p2p_peers_per_protocol`       | GaugeVec   | `protocol` | `AddPeer`++ / `RemovePeer`--                        | Peer mix by GS protocol         |
| `optimum_gossip-p2p_received_messages_count`  | CounterVec | `topic`    | `ValidateMessage` & `DuplicateMessage`              | All received (incl. duplicates) |
| `optimum_gossip-p2p_delivered_messages_count` | CounterVec | `topic`    | `DeliverMessage`                                    | Validated-delivered count       |
| `optimum_gossip-p2p_received_messages_bytes`  | CounterVec | `topic`    | Size of payload in `ValidateMessage`/`Duplicate...` | Ingest bytes                    |
| `optimum_gossip-p2p_delivered_messages_bytes` | CounterVec | `topic`    | Size of payload in `DeliverMessage`                 | Delivered bytes                 |
| `optimum_gossip-p2p_dropped_queue_full_total` | CounterVec | `topic`    | `RejectMessage` w/ `RejectValidationQueueFull`      | Backpressure (queue full)       |
| `optimum_gossip-p2p_dropped_throttled_total`  | CounterVec | `topic`    | `RejectMessage` w/ `RejectValidationThrottled`      | Backpressure (throttle)         |
| `optimum_gossip-p2p_dropped_rejected_total`   | CounterVec | `topic`    | `RejectMessage` (default branch)                    | Other rejections                |

## mump2p (Optimum RLNC pubsub)

| Metric Name                               | Type       | Labels     | How It’s Calculated                                 | Goal                                   |
| ----------------------------------------- | ---------- | ---------- | --------------------------------------------------- | -------------------------------------- |
| `optimum_mump2p_total_peers`              | Gauge      | none       | `AddPeer`++ / `RemovePeer`--                        | Total mumP2P peers                     |
| `optimum_mump2p_peers_per_protocol`       | GaugeVec   | `protocol` | `AddPeer`++ / `RemovePeer`--                        | Peer mix by mumP2P protocol            |
| `optimum_mump2p_received_messages_count`  | CounterVec | `topic`    | `ValidateMessage` & `DuplicateMessage`              | All received (incl. duplicates)        |
| `optimum_mump2p_delivered_messages_count` | CounterVec | `topic`    | `DeliverMessage`                                    | Validated-delivered count              |
| `optimum_mump2p_received_messages_bytes`  | CounterVec | `topic`    | Size of payload in `ValidateMessage`/`Duplicate...` | Ingest bytes                           |
| `optimum_mump2p_delivered_messages_bytes` | CounterVec | `topic`    | Size of payload in `DeliverMessage`                 | Delivered bytes                        |
| `optimum_mump2p_dropped_queue_full_total` | CounterVec | `topic`    | `RejectMessage` w/ `RejectValidationQueueFull`      | Backpressure (queue full)              |
| `optimum_mump2p_dropped_throttled_total`  | CounterVec | `topic`    | `RejectMessage` w/ `RejectValidationThrottled`      | Backpressure (throttle)                |
| `optimum_mump2p_dropped_rejected_total`   | CounterVec | `topic`    | `RejectMessage` (default branch)                    | Other rejections                       |
| `optimum_mump2p_shards_total`             | Counter    | none       | `AddTotalShardCount()`                              | Shards processed                       |
| `optimum_mump2p_shards_duplicate_total`   | Counter    | none       | `AddDuplicateShardCount()`                          | Duplicate shards                       |
| `optimum_mump2p_shards_unnecessary_total` | Counter    | none       | `AddUnnecessaryShardCount()`                        | Unnecessary shards (already decodable) |
| `optimum_mump2p_shards_unhelpful_total`   | Counter    | none       | `AddUnhelpfulShardCount()`                          | Unhelpful (linearly dependent) shards  |

## Generic P2P (app-level)

| Metric Name                            | Type       | Labels               | How It’s Calculated                    | Goal                                |
| -------------------------------------- | ---------- | -------------------- | -------------------------------------- | ----------------------------------- |
| `optimum_p2p_messages_published_total` | CounterVec | `topic`              | `IncP2PMessagesPublished(topic)`       | Outbound pub volume by topic        |
| `optimum_p2p_messages_received_total`  | CounterVec | `topic`,`peer_id`    | `IncP2PMessagesReceived(topic,peerID)` | Inbound msgs by topic & peer        |
| `optimum_p2p_message_size_bytes`       | HistVec    | `topic`              | `ObserveP2PMessageSize(topic,size)`    | Published message size distribution |
| `optimum_p2p_active_topics`            | Gauge      | none                 | `SetP2PActiveTopics(count)`            | Current # of subscribed topics      |
| `optimum_p2p_publish_errors_total`     | CounterVec | `topic`,`error_type` | `IncP2PPublishError(topic,errType)`    | Publish failure reasons             |

## gRPC I/O (per subsystem)

Two identical families, keyed by subsystem:

* `grpc_proxy_*` (use with `GRPCSubsystemProxy`)
* `grpc_p2p_*` (use with `GRPCSubsystemP2P`)

| Metric Name                                        | Type       | Labels (`method`) | How It’s Calculated                              | Goal                     |
| -------------------------------------------------- | ---------- | ----------------- | ------------------------------------------------ | ------------------------ |
| `optimum_grpc_<subsystem>_received_bytes_total`    | CounterVec | `method`          | `IncreaseGrpcReceived(subsys, method, msg)` size | Inbound bytes over gRPC  |
| `optimum_grpc_<subsystem>_messages_received_total` | CounterVec | `method`          | Same as above (count)                            | Inbound messages         |
| `optimum_grpc_<subsystem>_sent_bytes_total`        | CounterVec | `method`          | `IncreaseGrpcSent(subsys, method, msg)` size     | Outbound bytes over gRPC |
| `optimum_grpc_<subsystem>_messages_sent_total`     | CounterVec | `method`          | Same as above (count)                            | Outbound messages        |

## Gateway / Proxy

These are registered directly with `prometheus.New*` and thus do not have the `optimum_` prefix. They still inherit `cluster_id` and `protocol_type`.

| Metric Name                    | Type       | Labels                           | How It’s Calculated                     | Goal                                            |
| ------------------------------ | ---------- | -------------------------------- | --------------------------------------- | ----------------------------------------------- |
| `client_topic_threshold`       | GaugeVec   | `client_id`,`topic`              | `RecordClientTopicThreshold`            | Track per-client desired threshold              |
| `node_received_messages_total` | CounterVec | `node_id`,`topic`                | `RecordNodeReceivedMessage`             | Per-node ingestion counters                     |
| `message_seen_nodes_count`     | GaugeVec   | `message_id`,`topic`             | `RecordMessageSeenNodeCount`            | Distinct nodes that saw a message               |
| `message_first_seen_timestamp` | GaugeVec   | `message_id`,`topic`             | `RecordMessageFirstSeen` (Unix seconds) | First-seen timing for a message                 |
| `nodes_active_total`           | Gauge      | none                             | `UpdateActiveNodeCount`                 | Currently active nodes in proxy                 |
| `fallback_deliveries_total`    | CounterVec | `client_id`,`topic`              | `IncFallbackDeliveries`                 | Deliveries made via fallback path               |
| `message_size_bytes`           | HistVec    | `topic`,`client_id`              | `ObserveMessageSize`                    | Published size distribution (64B..32KB buckets) |
| `publish_error_total`          | CounterVec | `topic`,`client_id`,`error_type` | `IncPublishError`                       | Publish errors by cause                         |

And the `optimum_proxy_*` metrics:

| Metric Name                               | Type       | Labels              | How It’s Calculated                            | Goal                           |
| ----------------------------------------- | ---------- | ------------------- | ---------------------------------------------- | ------------------------------ |
| `optimum_proxy_p2p_connections`           | Gauge      | none                | `HandleNewProxyP2PConnection` / `...Closed...` | Active P2P conns to proxy      |
| `optimum_proxy_ws_clients_active`         | Gauge      | none                | `IncWSClients` / `DecWSClients`                | Active WebSocket clients       |
| `optimum_proxy_published_by_client_total` | CounterVec | `topic`,`client_id` | `IncPublishedByClient`                         | Client publish volume by topic |
