# Closure — mqttwire

Package: `server`. File: `mqtt.go`.

Removed (18 funcs stubbed): `mqttReader.reset`, `mqttReader.hasMore`,
`mqttReader.readByte`, `mqttReader.readPacketLen`, `mqttReader.readVarInt`,
`mqttReader.readString`, `mqttReader.readBytes`, `mqttReader.readUint16`,
`mqttWriter.WriteUint16`, `mqttWriter.WriteString`, `mqttWriter.WriteBytes`,
`mqttWriter.WriteVarInt`, `newMQTTWriter`, `mqttCheckFixedHeaderFlags`,
`mqttCheckRemainingLength`, `mqttParsePIPacket`, `mqttGetQoS`,
`mqttIsRetained`.

Kept: `mqttReader`/`mqttWriter` structs and fields (`buf`, `pos`, `pstart`,
`pbuf`; embedded `bytes.Buffer`), all `mqttPacket*`/`mqttPubFlag*`/
`mqttPacketMask`/`mqttPacketFlagMask` constants, `errMQTTMalformedVarInt`,
`errMQTTPacketIdentifierIsZero`, `ErrMaxPayload`, `mqttParse` and every
packet processor (which call the excised helpers), `copyBytes`.
Import `encoding/binary` blanked.

Tests snipped in `server/mqtt_test.go`: `TestMQTTReader`, `TestMQTTWriter`,
`TestMQTTPacketLenMaxPayloadViolation`,
`TestMQTTIncompleteConnectMaxPayloadViolationDisconnects`,
`TestMQTTMalformedFixedHeaderFlagsCauseDisconnect`,
`TestMQTTMalformedRemainingLengthCausesDisconnect`. Other MQTT tests drive
the codec through live connections and panic under excision — left in
place.
