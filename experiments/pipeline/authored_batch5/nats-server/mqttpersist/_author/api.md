# Exported API — mqttpersist

Package `server` (module `example.internal/msgkit/v2`) — persistence
encoding for MQTT retained messages: how a `mqttRetainedMsg` is stored as
a NATS headered message in the `$MQTT.rmsgs.` stream and read back,
including the legacy JSON fallback.

Surface (mqtt.go):
- `mqttEncodeRetainedMessage(rm *mqttRetainedMsg) (natsMsg []byte,
  headerLen int)` — encode to `NATS/1.0` header block + payload; returns
  header length for payload extraction.
- `mqttSliceHeaders(headers map[string][]byte, hdr []byte)` — fill a
  caller-supplied key-interest map with borrowed value slices.
- `mqttDecodeRetainedMessage(subject string, h, m []byte)
  (*mqttRetainedMsg, error)` — header decode + legacy JSON fallback +
  flag validation + subject→topic reconstruction.

Header keys (visible): `Nmqtt-RTopic`, `Nmqtt-ROrigin`, `Nmqtt-RFlags`,
`Nmqtt-RSource`; delete marker byte `-`; `mqttPacketFlagMask=0x0f`;
errors `errMQTTInvalidRetainFlags`, `errMQTTInvalidRetainedMessage`.

Callers: retained-message write path (`mqttEncodeRetainedMessage` before
JS store), `processRetainedMsg`/`loadRetainedMessages`/
`serializeRetainedMsgsForSub` on the read side. `natsSubjectStrToMQTTTopic`
and `mqttGetQoS` are retained helpers.
