# Exported API — mqtttopics

Package `server` (module `example.internal/msgkit/v2`) — bidirectional
MQTT-topic ↔ Msgkit-subject conversion and topic validation.

Surface: `mqttTopicToNATSPubSubject(mt []byte) ([]byte, error)` (PUBLISH
topic names, no wildcards allowed), `mqttFilterToNATSSubject(filter []byte)
([]byte, error)` (SUBSCRIBE filters, `+`/`#` wildcards allowed),
`mqttToNATSSubjectConversion(mt []byte, wcOk bool) ([]byte, error)` (shared
engine), `natsSubjectStrToMQTTTopic(string) []byte` /
`natsSubjectToMQTTTopic([]byte) []byte` (reverse direction for outbound
delivery), `mqttNeedSubForLevelUp(subject string) bool`,
`mqttValidateTopic(topic []byte, field string) error`,
`mqttValidateString(value, field string) error`,
`isMQTTReservedSubscription(subject string) bool`,
`mqttMustIgnoreForReservedSub(sub *subscription, subject string) bool`,
`sparkbParseBirthDeathTopic(topic []byte) (isBirth, isDeath, isCertificate
bool)`, `sparkbReplaceDeathTimestamp(msg []byte) []byte`.

Callers: `mqttParsePub`, `mqttParseSubsOrUnsubs`, `mqttProcessSubs`,
delivery callbacks, `mqttHandleWill`, retained-message restore paths.
