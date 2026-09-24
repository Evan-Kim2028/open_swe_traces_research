# Details — protoparse

1. Op dispatch is case-insensitive and letter-by-letter; the FIRST byte is
   remembered in `c.op` so later arg handling can tell `RS+`/`RMSG` from
   `LS+`/`LMSG`. `R*` and `A*` ops are protocol errors from `CLIENT` kind
   connections; `L*` ops are protocol errors unless kind is `LEAF` or
   `ROUTER` (so `GATEWAY` may send `A+`/`A-` and `RS*`/`RMSG` but never
   `L*`). Inferable: partially — the `OP_*` constants and `c.op` field are
   visible, the per-kind gates are body-internal.
2. `PUB`/`HPUB`/`HMSG`/`SUB`/`UNSUB`/`ASUB`/`AUSUB`/`MSG` require at least
   one space or tab after the op token, then skip any further spaces/tabs
   before the arg. `CONNECT`, `INFO`, and `-ERR`'s arg behave differently:
   `CONNECT`/`INFO` skip leading spaces but do NOT require one
   (`CONNECT{...}\n` is legal), while `-ERR` REQUIRES a space/tab.
   Inferable: partially — visible in the state names, not the
   required-vs-optional split.
3. `PING`, `PONG`, and `+OK` terminate on `\n` ONLY and silently absorb any
   other byte before it (`PING garbage\r\n` is accepted). Inferable: no —
   the missing `default:` arm is the whole rule.
4. Inside every `*_ARG` state, `\r` sets `drop=1` and only the byte
   immediately before `\n` is trimmed from the assembled arg; a `\r` mid-arg
   stays in the arg. `\r` bytes are never appended to `argBuf`.
   Inferable: no.
5. Args are sliced directly from the read buffer (`buf[as:i-drop]`); only
   when an arg spans buffers is `argBuf` (backed by the fixed `scratch`
   array) allocated and fed `buf[as:i-drop]` at buffer end — a dangling
   trailing `\r` is excluded, so a `\r`/`\n` split across buffers still
   terminates cleanly. Mid-buffer bytes are appended to `argBuf` only AFTER
   it exists. Inferable: no.
6. `overMaxControlLineLimit` compares `len(arg)` to `mcl` for `CLIENT` kind
   but to `mcl*16` (int64-widened) for `LEAF`/`ROUTER`/`GATEWAY`; on excess
   it sends the error, closes the connection with
   `MaxControlLineExceeded`, and returns `ErrMaxControlLine`. It runs both
   when a line completes AND at every buffer boundary on the partial
   `argBuf` (deliberately approximate early enforcement). Inferable:
   partially — the comment documents the ×16 bound for non-clients, the
   int64 widening and the two check sites are body-internal.
7. After a `PUB`/`HPUB`/`MSG`/`HMSG` arg line, `pa.size` is the payload
   length WITHOUT the trailing CRLF, and the parser jumps the index to
   `as + pa.size - LEN_CR_LF` so only the boundary bytes are examined;
   `MSG_END_R`/`MSG_END_N` then demand exactly `\r` then `\n`. The complete
   `msgBuf` handed to `processInboundMsg` INCLUDES the trailing CRLF.
   Inferable: partially — the jump expression is visible in shape, the
   size-excludes-CRLF/buffer-includes-CRLF convention is not.
8. The post-arg jump is conditional on `msgBuf == nil` for `PUB_ARG` and
   `HPUB_ARG` but unconditional for `MSG_ARG` and `HMSG_ARG`.
   Inferable: no.
9. Split payload: at buffer end in `MSG_PAYLOAD`/`MSG_END_R`/`MSG_END_N`
   with `msgBuf == nil`, `clonePubArg` re-parses the saved arg out of
   `scratch` so `pa.*` slices point into `argBuf`, not the dead read buffer;
   then `msgBuf` is either carved out of `scratch` after `argBuf` (when
   `pa.size <= cap(scratch) - len(argBuf)`) or freshly allocated with
   capacity `pa.size + LEN_CR_LF`. If the leftover bytes `len(buf[as:])`
   exceed `pa.size + LEN_CR_LF` it is a protocol error (rogue-client guard).
   Inferable: no.
10. `clonePubArg` re-dispatches by kind: `ROUTER`/`GATEWAY` honour `lmsg`
    (origin-cluster args) then `pa.hdr < 0` chooses routed vs routed-header;
    `LEAF` chooses leaf vs leaf-header; other kinds choose `processPub` vs
    `processHeaderPub` (the latter with `nil` remaining). Inferable: no.
11. While the auth gate is set, only `CONNECT` may proceed for `CLIENT`
    unless `NoAuthUser` (or the websocket-specific `NoAuthUser` override)
    names an existing user that is not `ProxyRequired` and whose
    `AllowedConnectionTypes` admit this connection — then the auth timer is
    cleared and `connectReceived` set BEFORE `checkAuthentication` runs
    (ordering matters: auth may install an expiry timer in the same slot).
    `LEAF` may instead send `INFO` first only when leaf compression is
    negotiated. Any other first op is `authViolation` + `ErrAuthentication`,
    not a parse error. `authSet` is re-snapshotted after a `CONNECT` is
    processed. Inferable: partially — the comments sketch the rule, the
    exact ordering and per-kind conditions are body-internal.
12. `MSG_ARG`/`HMSG_ARG` dispatch only for `ROUTER`/`GATEWAY` (`RMSG` vs
    `LMSG` via `c.op`, `LMSG` also setting `lmsg`) and `LEAF`; the `CLIENT`
    arm does nothing. `SUB_ARG`/`UNSUB_ARG` route per kind with `c.op`
    selecting remote vs leaf semantics for `ROUTER`. Inferable: no.
13. `protoSnippet` returns a `%q`-quoted slice `buf[start:stop]` where
    `stop = start+max` is clamped to `len(buf)-1` — one byte SHORT of the
    end — and `start >= len(buf)` yields the literal `""` (two quote
    chars). Inferable: no — the off-by-one clamp is an arbitrary choice.
14. `getHeader` lazily MIME-parses `msgBuf[0:pa.hdr]` only when `pa.hdr >
    0`, skips the first line (the `NATS/x.y` version line) before
    `ReadMIMEHeader`, caches the result in `ps.header`, and leaves the
    cache nil on parse error. Inferable: partially — the lazy-cache shape
    is visible, the skip-first-line and error-swallowing are not.
15. On message completion the parser resets `argBuf`/`msgBuf`/`header`,
    clears every `pa` field (`hdr` back to `-1`, `size` 0, `delivered`
    false, `trace` nil), clears `lmsg`, and returns to `OP_START` with
    `as = i+1`. `pa.hdr` is set to `-1` at `PUB_SPC`/`MSG_SPC` and `0` at
    `HPUB_SPC`/`HMSG_SPC`. Inferable: partially.
16. Parse errors `sendErr` an unknown-protocol-operation notice and return
    an error carrying the connection kind string, numeric state, index, and
    a `protoSnippet` excerpt of at most `PROTO_SNIPPET_SIZE` bytes.
    Inferable: partially — error text shape, not spelling.
