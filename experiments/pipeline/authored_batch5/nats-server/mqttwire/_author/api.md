# Exported API — mqttwire

Package `server` (module `example.internal/msgkit/v2`) — MQTT v3.1.1/v5 wire
codec primitives used by `client.mqttParse` and every MQTT writer path.

Surface: `mqttReader` (cursor over a read buffer with `pstart`/`pbuf` for
partial-packet carry-over) — `reset(buf)`, `hasMore()`, `readByte(field)`,
`readUint16(field)`, `readBytes(field, cp)`, `readString(field)`,
`readVarInt() (v int, complete bool, err error)`,
`readPacketLen(maxLen int32) (v int, complete bool, err error)`; `mqttWriter`
(embeds `bytes.Buffer`) — `newMQTTWriter(cap)`, `WriteUint16`,
`WriteString`, `WriteBytes`, `WriteVarInt`. Validators:
`mqttCheckFixedHeaderFlags(packetType, flags byte) error`,
`mqttCheckRemainingLength(packetType byte, pl int) error`,
`mqttParsePIPacket(r *mqttReader) (uint16, error)`, `mqttGetQoS(flags) byte`,
`mqttIsRetained(flags) bool`.

Callers: `mqttParse`, `mqttParseConnect`, `mqttParsePub`,
`mqttParseSubsOrUnsubs`, `mqttEnqueue*` writers, test helpers in
`mqtt_test.go`/`auth_callout_test.go`.
