# Details — mqtttopics

1. MQTT→subject conversion (`mqttToNATSSubjectConversion`): `/` becomes `.`,
   `.` becomes `//`, `+` becomes `*`, `#` becomes `>` — but a leading `/`
   produces `/.` (i.e. an empty first level `"/"` is preserved by inserting
   the separator AFTER it), an interior `//` becomes `/./` (empty level kept),
   and a trailing `/` produces `/.` likewise (empty last level kept).
   Inferable: partially — the doc comment on the function describes the `/`
   rules but the empty-level preservation cases are worked only in the body.
2. A `.` in an MQTT topic expands to `//` (two level seps) so that
   nats-side `.`-splitting can round-trip the literal; the reverse mapping
   `natsSubjectToMQTTTopic` collapses `//` back to `.` and `/.` pairs back
   to `/` — implemented as a left-to-right scan where a `/` followed by `.`
   or `/` consumes BOTH characters. Inferable: no — the two-character
   lookahead pairing is body-internal.
3. Whitespace (` ` `\t` `\n` `\r` `\f`) and DEL (0x7f) anywhere in a topic or
   filter reject with `errMQTTUnsupportedCharacters` — even though MQTT
   itself allows them — because the converted subject would corrupt the
   wire control line and DEL is a reserved pivot in the subject tree.
   Inferable: doc — comments state both reasons; the exact rune set is
   body-internal.
4. Wildcards are rejected in publish topics (`wcOk=false`) with an error
   naming the topic; in filters they map to `*`/`>` at ANY position —
   including positions MQTT would consider invalid (e.g. `foo/+bar`,
   `a/#/b` are converted, not rejected). Inferable: partially — the
   wcOk gate is visible, the no-position-validation choice is not.
5. When nothing needs converting the input slice is returned UNALLOCATED
   (same backing array); conversion allocates lazily on the first byte
   that needs rewriting. Inferable: doc — the comment says "no memory is
   allocated" but callers can rely on the identity.
6. `mqttNeedSubForLevelUp` is true iff the subject is at least 3 chars and
   ends with `.>` — a multi-level wildcard needing an extra subscription
   one level up. Inferable: partially — the comment states the rule.
7. `mqttValidateTopic`/`mqttValidateString` reject only invalid UTF-8 and
   embedded NUL; empty values and wildcards pass topic validation (wildcard
   legality is the converter's job, not the validator's). Inferable: doc.
8. `isMQTTReservedSubscription` matches exactly `#`, `*`, or `*.` prefix —
   i.e. subscriptions that would catch `$SYS`-style reserved subjects.
   `mqttMustIgnoreForReservedSub` drops deliveries of `$`-prefixed subjects
   to those subscriptions only. Inferable: partially — the `sub.mqtt.
   reserved` flag and comment sketch it; the `*` + `.` spelling is subtle.
9. `sparkbParseBirthDeathTopic` recognises `spBv1.0/<group>/<type>/…` topics
   with 3 or 4 `/`-separated parts after the (optional
   `$sparkplug/certificates/`) prefix, and only the NBIRTH/DBIRTH/NDEATH/
   DDEATH type tokens count as birth/death. Inferable: no — the
   certificates-prefix handling and part-count window are arbitrary.
10. `sparkbReplaceDeathTimestamp` scans protobuf fields; on field 1
    (timestamp) it substitutes the current time, on scan error returns the
    ORIGINAL buffer, and appends a timestamp field if none was present.
    Inferable: no.
