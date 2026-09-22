# Details — mqttpersist

1. Encoded record = `NATS/1.0\r\n` header block then raw payload. Header
   order: `Nmqtt-RTopic`, `Nmqtt-RFlags`, optional `Nmqtt-ROrigin`,
   optional `Nmqtt-RSource`, blank line. Flags render as LOWERCASE hex via
   `strconv.FormatUint(flags, 16)`. Inferable: partially — order/hex is
   wire-visible only in the encoder.
2. A retained-delete record (empty Msg) emits a `-` byte BEFORE the hex
   flags in `Nmqtt-RFlags`, and contributes +1 to the precomputed length.
   Inferable: partially — delete marker constant visible.
3. `mqttSliceHeaders` parses lines after the status line: key ends at the
   first `:`; spaces BETWEEN key and `:` are walked back out; an empty
   resulting key stops the parse; value runs to the next CRLF with leading
   spaces stripped; the stored slice is capacity-limited
   (`hdr[i:j:j]`); only keys already present in the caller's map are
   recorded. Inferable: partially — trim/backtrack rules test-covered.
4. `mqttDecodeRetainedMessage` slices ONLY the three headers it needs
   (Topic is reconstructed, not read). If `Nmqtt-RFlags` is absent it
   falls back to `json.Unmarshal(m, &rm)` (legacy JSON records) — a nil
   result → `errMQTTInvalidRetainedMessage`. Inferable: doc — comment.
5. Flags header: a leading `-` delete marker is stripped, remainder parsed
   base-16 8-bit; parse failure → `errMQTTInvalidRetainFlags`; decoded
   Flags >= `mqttPacketFlagMask` or QoS > 2 → `errMQTTInvalidRetainFlags`.
   Inferable: partially.
6. `rm.Subject` = `subject` minus the `$MQTT.rmsgs.` prefix; `rm.Topic` =
   `natsSubjectStrToMQTTTopic(rm.Subject)` — the topic is always
   reconstructed from the stream subject, never trusted from headers.
   Inferable: doc — comment explains the subject-transform rationale.
7. `rm.Msg` borrows `m` (no copy) — documented caller-copy contract.
   Inferable: doc — function comment.
