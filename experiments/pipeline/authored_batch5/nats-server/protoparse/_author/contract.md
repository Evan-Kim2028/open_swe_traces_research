# Contract — protoparse

Inbound wire-protocol parser: a byte-oriented state machine covering op
dispatch, arg slicing, payload boundaries, and per-kind gates. Every
commitment below is covered by a hidden test; every hidden test maps to
a commitment.

## Commitments

1. **Op dispatch per kind.** Ops match case-insensitively; client
   connections reject routed/leaf/account ops; gateways never accept
   leaf ops; routes and leaves accept their own op families. Covered by
   `TestDetail01`.
2. **Op-arg separator.** Pub/sub/msg-style ops require a space or tab
   after the op token and skip further separators; connect/info do not
   require one; the error op requires one. Covered by `TestDetail02`.
3. **Fixed-line ops.** Ping/pong/ok lines terminate on newline and
   absorb any preceding bytes. Covered by `TestDetail03`.
4. **CR handling in args.** A mid-arg carriage return stays in the
   assembled arg; a dangling trailing CR at a buffer boundary is
   excluded. Covered by `TestDetail04`.
5. **Arg spanning buffers.** An arg that crosses a read boundary is
   assembled in the scratch buffer and still parses; a CR/LF split
   across buffers terminates cleanly. Covered by `TestDetail05`.
6. **Control-line limit.** The limit compares the arg length against
   the configured maximum for clients and sixteen times that for other
   kinds; excess returns the max-control-line error. Covered by
   `TestDetail06`.
7. **Payload boundary.** The parsed payload size excludes the trailing
   CRLF and the terminator states demand exactly CR then LF. Covered by
   `TestDetail07`.
8. **Conditional post-arg jump.** With no message buffer yet, the
   parser jumps into the payload and resumes correctly across a buffer
   split, then accepts the next op. Covered by `TestDetail08`.
9. **Arg re-parse on split.** When both arg and payload span buffers,
   the saved arg is re-parsed and the message completes. Covered by
   `TestDetail09`.
10. **Per-kind completion dispatch.** A split message on a leaf
    connection completes through the leaf path. Covered by
    `TestDetail10`.
11. **Auth gate.** While awaiting authentication, non-connect ops fail
    with the authentication error; connect lifts the gate. Covered by
    `TestDetail11`.
12. **Client msg rejection.** A client connection receiving a
    server-bound msg op gets a protocol error. Covered by
    `TestDetail12`.
13. **Error snippet.** The snippet helper returns a quoted slice of at
    most the requested size, clamped one byte short of the buffer end,
    and an empty quoted string for out-of-range starts. Covered by
    `TestDetail13`.
14. **Lazy header parse.** Headers are MIME-parsed only when a header
    size is recorded, the status line is skipped, the result is cached,
    and parse errors leave the cache nil. Covered by `TestDetail14`.
15. **Post-message reset.** On completion the parser clears the arg
    and message buffers, the cached header, and every parsed-arg field,
    and returns to the start state. Covered by `TestDetail15`.
16. **Error shape.** Parse errors carry the connection kind, numeric
    state and index, and a bounded quoted excerpt. Covered by
    `TestDetail16`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | partially — op gates asserted via accept/reject, not internals |
| TestDetail02 | 2 | partially — required-vs-optional separator split |
| TestDetail03 | 3 | no — shape: garbage before newline is absorbed |
| TestDetail04 | 4 | no — shape: mid-arg CR retained, dangling CR dropped |
| TestDetail05 | 5 | no — shape: split-arg and split-CRLF parses succeed |
| TestDetail06 | 6 | partially — ×16 bound asserted via limit calls |
| TestDetail07 | 7 | partially — boundary enforcement via accept/reject |
| TestDetail08 | 8 | no — shape: split payload then next op completes |
| TestDetail09 | 9 | no — shape: arg+payload split completes |
| TestDetail10 | 10 | no — shape: leaf split-msg completes |
| TestDetail11 | 11 | partially — gate behavior via flags; ordering details not pinned |
| TestDetail12 | 12 | no — shape: client MSG rejected |
| TestDetail13 | 13 | no — committed clamp rule asserted, literals derived from it |
| TestDetail14 | 14 | partially — lazy/cache/skip-line/error-nil asserted |
| TestDetail15 | 15 | partially — observable reset fields asserted |
| TestDetail16 | 16 | partially — error carries kind/state/index/bounded snippet |
