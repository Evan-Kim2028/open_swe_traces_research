# Details — capability

1. Wire form is space-separated tokens: a bare `name` for flag capabilities, `name=value` for
   valued ones; a capability holding N values emits one `name=value` token PER value.
   Inferable: partially.
2. The encoded order is first-insertion order — never sorted — and re-adding an existing name
   appends values without moving it. Inferable: doc — `String`'s doc comment says so.
3. `Set` on an existing capability replaces its values in place but keeps its position in the
   order; on a new name it appends. Inferable: partially.
4. `Add` on an existing name with NO values leaves its current values untouched — it does not
   clear them. Inferable: no.
5. Decoding `name=` (equals sign, empty right side) produces ONE value: the empty string —
   distinguishable from a bare `name` which produces none. Inferable: no.
6. Decode tolerates leading/trailing runs of space — empty chunks are skipped, never turned
   into empty-named entries. Inferable: partially.
7. `All` on an empty list returns nil (not an empty slice); encoding an empty or nil list
   produces no bytes and no error. Inferable: no.
8. `Delete` removes the name from both the lookup map and the insertion order — re-adding it
   afterwards puts it at the END, not its old slot. Inferable: no.
9. Validation fails on an empty argument value BEFORE checking whether the name is known —
   `bogus=` reports the empty-argument error, not the unknown-capability error. Inferable: no.
10. A fixed subset of capabilities REQUIRES an argument (agent, push-cert, symref,
    object-format, session-id) and only symref may carry more than one value — argument on a
    flag capability, missing required argument, and multi-value on a single-value capability
    are three distinct failures. Inferable: no — the assignment tables are excised; the
    constants alone don't reveal the rules.
11. The session-id value is checked against printable ASCII: any byte ≤ 32 (INCLUDING space)
    or ≥ 127 fails, and the value must fit in one maximum pkt-line payload. Inferable: no.
12. The default agent string is `go-git/6.x`; a `GO_GIT_USER_AGENT_EXTRA` environment value is
    appended after a space — but only when the value is non-blank after trimming, so a
    whitespace-only override is ignored. Inferable: no.
