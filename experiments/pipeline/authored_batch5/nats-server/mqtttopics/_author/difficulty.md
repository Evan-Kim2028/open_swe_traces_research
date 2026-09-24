# Difficulty — mqtttopics

predicted_flip: L2
details: 10

Missed edges: empty-level preservation (`/`→`/.`, `//`→`/./`, trailing
`/`→`/.`); `.`→`//` expansion and its 2-char lookahead collapse on the way
back; whitespace+DEL rejection beyond the MQTT spec; wildcards accepted at
any position in filters (no MQTT legality check); zero-copy when no rewrite
needed; reserved-sub matching on `*` + `.` spelling; sparkb certificate
prefix + 3–4 part window + protobuf timestamp substitution with original
fallback.

Hardness driver: a two-directional byte-mapping function whose doc comment
lists the headline rules but whose empty-level and lookahead cases live only
in the code.
