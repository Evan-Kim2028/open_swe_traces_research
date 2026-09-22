# API left after excision

```go
func encodeConsumerState(state *ConsumerState) []byte
```

Encodes `ConsumerState` (ack floor, delivered sequences, pending map, redelivered map) into the on-disk / RAFT consumer-state blob. `hdrLen`, `magic`, `seqsHdrSize`, and `binary.PutUvarint`/`PutVarint` remain available in the package.
