# Details — hdrsurgery

1. `getHeaderKeyIndex` finds the key via `bytes.Index` but REJECTS a match
   unless it is preceded by a full CRLF (`hdr[i-1]=='\n'` AND
   `hdr[i-2]=='\r'`, so the key can never occupy index 0..1) and is
   immediately followed by `:`. On rejection it retries from
   `index+keyLen`, so a longer key that merely STARTS with the searched
   key (e.g. JSExpectedLastSubjSeq vs ...SubjSeqSubj) does not shadow the
   real one. Inferable: partially — prefix-shadowing is test-covered.
2. `sliceHeader` returns `hdr[start:index:index]` — a slice INTO the
   header (no copy) whose capacity stops at the value end; skips leading
   spaces after `:`, value ends at the first CRLF. `getHeader` is
   sliceHeader + an exact-size copy. Inferable: doc — test asserts
   `cap(sliced)==2`.
3. `removeHeaderIfPresent` loops until no exact-key line remains (not just
   the first); a removed line spans key start through its CRLF; result
   `len <= len(emptyHdrLine)` → nil. Inferable: partially — duplicates
   case is test-covered.
4. `removeHeaderIfPrefixPresent` removes lines whose KEY begins with the
   prefix; the prefix occurrence must be preceded by `\n` and the line's
   value terminated by CRLF; non-line-start occurrences are skipped and
   scanning continues from `index+len(prefix)`. Inferable: partially.
5. `removeHeaderStatusIfPresent` only strips when hdr starts with
   `NATS/1.0` and a `\r` exists at index > len("NATS/1.0"); removing leaves
   exactly `emptyHdrLine` → nil. Inferable: no.
6. `isServiceReply`: `len>3 && reply[:4] == "_R_."`. `isJSAckSubject`:
   `len > jsAckPreLen && subject[:jsAckPreLen] == "$JS.ACK."`.
   Inferable: yes — constants visible.
7. `jsAckDeliverIdx` returns the index of the first `@` seen AFTER at
   least 8 `.` characters in the reply — stream/consumer/subject tokens
   may themselves contain `@`, and an `@` before the eighth dot does not
   count; non-JS-ack replies → -1. Inferable: doc — comment spells the
   format, test pins the 8-dot rule.
8. `isReservedReply` = isServiceReply || isJSAckSubject ||
   `hasGWRoutedReplyPrefix` (gateway helper, retained). Inferable: yes.
9. `splitSubjectQueue` splits on ANY whitespace run (strings.Fields),
   trims surrounding space; 1 field → subject only, 2 → subject+queue,
   0 or >2 → error; both tokens must satisfy `IsValidSubject`.
   Inferable: yes — table test.
10. `splitArg` splits on ` `, `\t`, `\r`, `\n` (runs collapse); no empty
    tokens; stack array capped at MAX_MSG_ARGS then grows by append.
    Inferable: yes.
