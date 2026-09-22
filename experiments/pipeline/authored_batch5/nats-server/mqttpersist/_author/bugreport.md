# Bug report — mqttpersist

Retained MQTT messages cannot be persisted or restored. The encoder
(`mqttEncodeRetainedMessage`), the header slicer (`mqttSliceHeaders`), and
the decoder (`mqttDecodeRetainedMessage`) panic with `excised`. Retained
publish fails on write and retained recovery/subscription delivery fails
on read.

Reproduce:

    go test ./server/ -run 'TestMQTTSliceHeadersAndDecodeRetainedMessage|TestMQTTPoisonedJSONRecords'

Restore the exact record format: header key order and names, hex flag
rendering, the `-` delete marker, space-tolerant header slicing with
capacity-limited values, the absent-flags JSON fallback, flag range/QoS
validation, and subject-derived topic reconstruction.

Work only from the repository and test output. Do not use web search or
any tool that accesses the internet. This repository is fully
self-contained; do NOT fetch upstream sources. Preserve all unrelated
tests.
