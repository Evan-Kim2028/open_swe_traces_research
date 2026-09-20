# Details — reportstatus

1. First packet line must be `unpack <status>` — a flush or any other first line (including an
   `ok`/`ng` line) is rejected. Inferable: partially.
2. Decode accepts ANY unpack status string without erroring; failure surfaces only later through
   the status object's error view — `unpack` not equal to `ok` becomes an unpack-status error
   carrying that status text. Inferable: no — the decode-then-check split is not obvious.
3. Per-ref lines are `ok <refname>` (exactly two fields) or `ng <refname> <reason>` (exactly
   three); `ok` with a trailing field or `ng` missing its reason is malformed.
   Inferable: no.
4. The status object's error view reports the unpack failure FIRST, then the first ref failure
   only — later ref failures are dropped even when several exist (mirrors canonical git).
   Inferable: partially.
5. A ref whose status string is exactly `ok` produces no error; any other status string —
   including the empty string — is a command-status error carrying the ref name and the status.
   Inferable: partially.
6. Decode stops at the first flush packet and requires it: running out of input before the flush
   is a missing-terminator error. Inferable: partially.
7. Encode emits `unpack <status>`, then one line per ref (`ok <ref>` or `ng <ref> <status>`),
   then a flush — nothing before `unpack` and nothing after the flush. Inferable: partially.
8. On encode a ref's line is `ok` exactly when its error view is nil — i.e. when its Status field
   is the literal `ok` — and `ng <ref> <status>` otherwise. Inferable: no.
9. Error values distinguish the two failure kinds: an unpack failure and a per-ref command
   failure are different error types, each carrying its offending status text (and ref name for
   the command case). Inferable: doc — both error types are kept in the source.
10. The first line is read before any command lines — a stream that starts with a ref line, or
    is empty, is an early EOF-style failure, not a ref-status decode. Inferable: partially.
