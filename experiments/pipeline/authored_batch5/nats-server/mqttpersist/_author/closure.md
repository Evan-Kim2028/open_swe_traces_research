# Closure — mqttpersist

Package `server`, file `server/mqtt.go`.

Removed (stubbed): `mqttEncodeRetainedMessage`, `mqttSliceHeaders`,
`mqttDecodeRetainedMessage`.

Retained: `mqttRetainedMsg` type, all `Nmqtt-R*` header constants,
`mqttRetainedFlagDelMarker`, `mqttPacketFlagMask`, `errMQTTInvalid*`,
`mqttGetQoS`, `natsSubjectStrToMQTTTopic`, `mqttRetainedMsgsStreamSubject`,
the session-manager and JS-store callers.

No import changes. Tests snipped:
`TestMQTTSliceHeadersAndDecodeRetainedMessage` (server/mqtt_test.go,
~160-line table test covering the slicer, decoder, encoder, delete marker,
JSON fallback and flag validation). No test files deleted;
`TestMQTTPoisonedJSONRecordsDoNotCrashServer` and retained-message
integration tests still exercise the closure.
