# Contract (L2) — ruinfo

`MakeRequestInfo` marks a request bypassed when its request source is the
internal-others source; a non-write request is not a write, while a
write-typed request with zero content still reports `IsWrite` with zero
write bytes. Write bytes track the content of `Prewrite` mutations and
`Commit` keys — more content produces a strictly larger count — while other
write-typed requests contribute nothing. `MakeResponseInfo` derives read
bytes from a coprocessor response's payload and from a scan response's
whole encoded size, and a present processed-versions-size detail overrides
the payload-derived count. KV CPU prefers the V2 nanosecond wall time,
then the earlier millisecond field, then the legacy detail. Nil and
unlisted responses produce an empty `ResponseInfo`, and `Succeed` is
unconditionally true.

| test | commitment |
| --- | --- |
| `TestDetail01` | The internal-others source marks bypass while an external source does not; a read request is not a write; an empty prewrite is a write with zero bytes. |
| `TestDetail02` | Prewrite write bytes are positive and strictly monotone in mutation content; commit write bytes track key bytes; an unlisted write-typed request counts nothing. |
| `TestDetail03` | Coprocessor read bytes cover the payload; scan read bytes cover the response content; a present processed-versions-size detail overrides the count. |
| `TestDetail04` | KV CPU takes the V2 nanosecond field when present, else the earlier millisecond field, else the legacy detail — each converted to a duration per its declared unit. |
| `TestDetail05` | Nil and unlisted responses yield an empty non-nil `ResponseInfo`, and `Succeed` is always true. |
