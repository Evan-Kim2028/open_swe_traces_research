# Contract — mqttpersist

Persistence encoding for MQTT retained messages: how a retained message
is stored as a headered NATS message and read back, including the legacy
JSON fallback. Every commitment below is covered by a hidden test; every
hidden test maps to a commitment.

## Commitments

1. **Wire layout.** The encoded record is a `NATS/1.0` header block
   followed by the raw payload. Header order is topic, flags, then the
   optional origin and source; flags render as lowercase hexadecimal.
   The returned header length splits header from payload. Covered by
   `TestDetail01`.
2. **Delete marker.** A record with an empty payload emits a `-` marker
   byte before the hex flags. Covered by `TestDetail02`.
3. **Header slicing.** A header line's key ends at the first colon with
   spaces before the colon walked back; an empty key ends the parse;
   values run to the next CRLF with leading spaces stripped; stored
   slices are capacity-limited; only keys present in the caller's map
   are recorded. Covered by `TestDetail03`.
4. **Decode + JSON fallback.** Decoding reads only the flag/origin/
   source headers it needs; when the flags header is absent the payload
   is decoded as a legacy JSON record, and a record that decodes to a
   nil message yields the invalid-retained-message error. Covered by
   `TestDetail04`.
5. **Flag validation.** A leading delete marker is stripped and the
   remainder parsed base-16; a parse failure, flags at or above the
   packet-flag mask, or a QoS above 2 yield the invalid-flags error.
   Covered by `TestDetail05`.
6. **Subject-derived topic.** The decoded subject is the stream subject
   minus its stream prefix, and the topic is reconstructed from that
   subject — never taken from a header. Covered by `TestDetail06`.
7. **Borrowed payload.** The decoded message references the caller's
   payload slice rather than copying it. Covered by `TestDetail07`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | partially — order, lowercase hex, blank-line termination |
| TestDetail02 | 2 | partially — marker presence; the +1 length is internal |
| TestDetail03 | 3 | partially — trim/backtrack/stop/interest rules |
| TestDetail04 | 4 | doc — fallback path and nil-result sentinel |
| TestDetail05 | 5 | partially — error identity per committed rule |
| TestDetail06 | 6 | doc |
| TestDetail07 | 7 | doc — asserted via aliasing |
