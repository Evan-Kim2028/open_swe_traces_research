# Details — leafmsg

1. Arg tokenisation is inline (not `splitArgs`): splits on space, tab,
   CR, LF into `c.argsa[:0]` — zero alloc. `c.pa.arg` is set to the raw
   arg BEFORE any arity check, so it is populated even on error.
   Inferable: partially — visible in a benchmark but not the signature.
2. `processLeafMsgArgs` layout: `subject [reply] size` for ≤3 args.
   With ≥4 args, `args[1]` becomes a single-byte reply INDICATOR:
   `+` means `args[2]` is the reply, `|` means no reply; anything else
   (or a multi-byte token) → "Bad or Missing Reply Indicator" error.
   Queues are the tokens between the reply position and the size
   (`args[3:len-1]` with reply, `args[2:len-1]` without). Inferable:
   partially — the `+`/`|` mini-protocol is leaf-specific internal.
3. `processLeafHeaderMsgArgs` adds `hdr`/`hdb`: needs ≥3 args, size is
   the LAST token and header size the second-to-last (`args[len-2]`);
   in the indicator form queues span `args[3:len-2]`/`args[2:len-2]`.
   Inferable: partially.
4. Both parsers set `szb`/`size` via `parseSize`; `size < 0` →
   "Bad or Missing Size" error (header variant also checks `hdr < 0`).
   Subject is assigned from `args[0]` only AFTER all validation passes.
   Inferable: partially — ordering is internal.
5. A parsed `size` exceeding the client's `mpay` (unless `jwt.NoLimit`)
   calls `maxPayloadViolation` and returns `ErrMaxPayload` — a typed
   sentinel error, not a parse error. Inferable: yes — TestParseLeafMsg
   BadSize asserts the sentinel.
6. `keyFromSub` emits `"subject"` or `"subject queue"` (single space).
   Inferable: yes.
7. `keyFromSubWithOrigin` prepends a one-byte kind prefix + space:
   `'L'` when `sub.origin` non-empty, `'N'` when `sub.leaf` and no
   origin, `'R'` otherwise; then subject, then ` queue` if set, then
   ` origin` only when an origin exists. All four forms are distinct —
   TestLeafNodeRoutedSubKeyDifferentBetweenLeafSubAndRoutedSub requires
   it. Inferable: partially — the byte choices and field order are an
   internal wire contract with route bookkeeping.
