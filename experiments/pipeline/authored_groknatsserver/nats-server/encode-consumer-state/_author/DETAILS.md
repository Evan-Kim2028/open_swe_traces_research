1. Byte 0 of the encoding is the file-store magic value 22. Inferable: no
2. Byte 1 of the encoding is version 2. Inferable: no
3. After the 2-byte header the encoder writes, in order, uvarints for ack-floor consumer seq, ack-floor stream seq, delivered consumer seq, delivered stream seq, then the pending-map length. Inferable: partially
4. When the pending map is empty the encoder does not write a base timestamp. Inferable: yes
5. When the pending map is non-empty the encoder writes a signed varint base timestamp equal to `time.Now().Round(time.Second).Unix()`, then one record per pending entry. Inferable: doc
6. Each pending record is three varints: stream-seq minus ack-floor stream, consumer-seq minus ack-floor consumer, and `baseTimestamp - (pending.Timestamp / 1e9)` (timestamp stored as whole seconds, inverted against the base). Inferable: no
7. After pending records the encoder always writes a uvarint redelivered-map length, even when zero. Inferable: doc
8. Each redelivered record is two uvarints: stream-seq minus ack-floor stream, then the redelivery count (not delta-encoded). Inferable: partially
9. Map iteration order is the Go map order; decoders must not assume a sort. Inferable: yes
10. The returned slice is truncated to the number of bytes actually written. Inferable: yes
11. If both maps are empty the encoder may use a stack buffer of `seqsHdrSize`; otherwise it allocates. Inferable: partially
