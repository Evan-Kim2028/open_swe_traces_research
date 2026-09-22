# Closure — mqtttopics

Package: `server`. File: `mqtt.go`.

Removed (12 funcs stubbed): `mqttTopicToNATSPubSubject`,
`mqttFilterToNATSSubject`, `mqttToNATSSubjectConversion`,
`natsSubjectStrToMQTTTopic`, `natsSubjectToMQTTTopic`,
`mqttNeedSubForLevelUp`, `mqttValidateTopic`, `mqttValidateString`,
`isMQTTReservedSubscription`, `mqttMustIgnoreForReservedSub`,
`sparkbParseBirthDeathTopic`, `sparkbReplaceDeathTimestamp`.

Kept: `mqttTopicLevelSep`/`btsep`/`pwc`/`fwc`/`mqttSingleLevelWC`/
`mqttMultiLevelWC`/`mqttReservedPre` constants, `errMQTTUnsupportedCharacters`,
`sparkb*` constants, `protoScanField`/`protoEncodeVarint` (proto.go),
`copyBytes`/`bytesToString`/`stringToBytes`, `subscription.mqtt.reserved`
field, and every caller (`mqttParsePub`, `mqttParseSubsOrUnsubs`,
`mqttProcessSubs`, delivery callbacks, sparkplug handling).
Import `unicode/utf8` blanked.

Tests snipped in `server/mqtt_test.go`: `TestMQTTTopicAndSubjectConversion`,
`TestMQTTFilterConversion`, `TestMQTTSubjectWildcardStart`,
`TestMQTTTopicWithDot`, `TestMQTTSparkbDeathHandling`,
`TestMQTTSparkbBirthHandling`, `TestMQTTPublishTopicErrors`.
