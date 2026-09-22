# Details — mqttwire

1. `readVarInt` decodes MQTT's variable-byte integer: each byte contributes
   `(b & 0x7f) * m` with `m` starting at 1 and multiplied by 0x80 per
   continuation; a byte with bit 0x80 clear terminates the value. It returns
   `(v, true, nil)` on completion, `(0, false, nil)` when the buffer is
   exhausted mid-value (caller retries with more data), and
   `errMQTTMalformedVarInt` once `m` would exceed `0x200000` — i.e. a 4-byte
   cap. Inferable: doc — MQTT's varint spec is well-known, but the
   complete-flag contract and the exact overflow point are body-internal.
2. `readPacketLen` reads a varint then checks `pos + v` against the buffer:
   when the packet is complete it returns `(v, true, nil)`; when incomplete
   (or when the varint itself was incomplete) it stashes `buf[pstart:]`
   into `r.pbuf` and returns `(0, false, nil)`. The `maxLen` check compares
   `packetLen = packetEnd - pstart` — the WHOLE packet including fixed
   header bytes — against `maxLen` unless `maxLen == jwt.NoLimit`, and on
   violation returns `ErrMaxPayload` WITHOUT touching `pbuf`. Inferable:
   no — the whole-packet-not-remainder accounting and pbuf-skip-on-error
   are arbitrary choices.
3. `reset` prepends any pending `pbuf` (partial packet left by a previous
   `readPacketLen`) before the new buffer, clears `pbuf`, and zeroes
   `pos`/`pstart`. Inferable: partially — the field names imply carry-over,
   the prepend-then-clear mechanics are internal.
4. `readBytes`/`readString` are uint16-length-prefixed; a zero length returns
   `nil`/`""` with no error and consumes only the 2-byte prefix; `cp=true`
   returns a copy (detaching from the read buffer) while `cp=false` aliases
   `buf`. `readUint16` is big-endian and short-buffers produce
   `io.ErrUnexpectedEOF`-wrapped errors; `readByte` produces
   `io.EOF`-wrapped errors; both include the field name. Inferable:
   partially — error shape (field name + wrapped sentinel) is inferable,
   the alias-vs-copy rule and the zero-length shortcut are not.
5. `mqttCheckFixedHeaderFlags`: packet types CONNECT, PUBACK, PUBREC,
   PUBCOMP, PINGREQ, DISCONNECT require flags == 0; PUBREL, SUBSCRIBE,
   UNSUBSCRIBE require flags == 0x2; PUBLISH and every other type accept
   anything (return nil). Inferable: doc — the table follows the MQTT spec,
   but the "unknown types pass" arm is a choice.
6. `mqttCheckRemainingLength`: CONNECT/PUBLISH/SUBSCRIBE/UNSUBSCRIBE accept
   any length; PUBACK/PUBREC/PUBREL/PUBCOMP require pl == 2; PINGREQ/
   DISCONNECT require pl == 0; unknown types pass. Inferable: doc —
   spec-driven table, same caveat on the default arm.
7. `mqttParsePIPacket` reads a uint16 packet identifier and rejects 0 with
   `errMQTTPacketIdentifierIsZero`. Inferable: doc — MQTT forbids PI=0.
8. `mqttGetQoS` extracts `(flags & 0x6) >> 1`; `mqttIsRetained` tests
   `flags & 0x1`. Inferable: doc — the `mqttPubFlag*` constants are visible.
9. `WriteUint16` emits big-endian; `WriteString`/`WriteBytes` prepend a
   uint16 length; `WriteVarInt` emits LSB-first continuation bytes
   (`0x80`-flagged) and always emits at least one byte even for 0.
   `newMQTTWriter(cap)` returns a `*mqttWriter` pre-grown to `cap`.
   Inferable: doc — standard MQTT encoding, no surprises beyond layout.
