# Contract — leafmsg

Leaf-node inbound message argument parsing (`LMSG`/`LHMSG`) and the
routed-subscription map keys. Every commitment below is covered by a
hidden test; every hidden test maps to a commitment.

## Commitments

1. **Inline tokenization.** The argument bytes are tokenized on space,
   tab, CR, and LF; the raw arg is stored on the parser state before any
   arity check so it is populated even when the parse fails. Covered by
   `TestDetail01`.
2. **Message layout.** The plain form is `subject [reply] size`. With
   four or more tokens the second token is a single-byte reply
   indicator: `+` means the next token is the reply, `|` means no reply;
   any other byte or a multi-byte token yields a reply-indicator error.
   Queue subjects are the tokens between the reply position and the
   size. Covered by `TestDetail02`.
3. **Header layout.** The header form takes at least three tokens; the
   total size is the last token and the header size the second-to-last.
   In indicator form the queue subjects sit between the reply position
   and those two trailing size tokens. Covered by `TestDetail03`.
4. **Size validation.** Sizes go through the shared size parser; a
   negative result produces a size error (and a negative header size a
   header-size error). The subject is assigned only after validation
   passes, while the raw size tokens are recorded regardless. Covered by
   `TestDetail04`.
5. **Max payload.** A parsed size above the client's maximum payload
   (unless unlimited) returns the typed max-payload sentinel error.
   Covered by `TestDetail05`.
6. **Plain key.** The routed-subscription key is the subject, or the
   subject and queue separated by a single space. Covered by
   `TestDetail06`.
7. **Origin key.** The collision-safe key prefixes a one-byte kind:
   leaf-with-origin, leaf-without-origin, or plain routed — then the
   subject, then the queue when set, then the origin when one exists.
   All four forms are mutually distinct for the same subject. Covered
   by `TestDetail07`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | partially — tokenization and early raw-arg store; alloc count not pinned |
| TestDetail02 | 2 | partially — layout, indicator bytes, and queue span |
| TestDetail03 | 3 | partially — header/hdb positions and shifted queue span |
| TestDetail04 | 4 | partially — error names and validation ordering |
| TestDetail05 | 5 | yes |
| TestDetail06 | 6 | yes |
| TestDetail07 | 7 | partially — kind bytes and field order per the visible comment |
