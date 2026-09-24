# Contract — sigcodec

`Signature` decode/encode, `DecodeObject` type dispatch, `Blob` adoption of
an encoded object, `ObjectIter` lazy decoding, and `GetBlob` type checking.
Every commitment below is covered by a hidden test; every hidden test maps
to a commitment.

## Commitments

1. **Bracket anchoring.** Signature parsing anchors on the last `<` and
   last `>` — angle brackets inside the name are retained, and a line
   with missing or inverted brackets leaves the signature's fields
   unchanged rather than zeroing them. Covered by `TestDetail01`.
2. **No interior validation.** The name is the text before the final `<`;
   the email is the raw interior of the brackets — no trimming or
   well-formedness check is applied. Covered by `TestDetail02`.
3. **Time decode gating (shape).** Timestamp and timezone are decoded only
   when at least one separator follows `>` — a signature with no time
   portion keeps `When` zero-valued. Asserted shape: presence/absence of
   the separator, not the offset width. Covered by `TestDetail03`.
4. **Negative zone sign (shape).** A negative timezone offset negates the
   whole offset — hours and minutes share the sign. Asserted shape:
   `-0230` differs in sign from `+0230` and in magnitude from `-0200`.
   Covered by `TestDetail04`.
5. **Pre-epoch clamp (shape).** Encoding a signature whose `When` is
   before the Unix epoch emits a non-negative timestamp rather than a
   negative serialisation. Asserted shape: the emitted field parses as a
   non-negative integer. Covered by `TestDetail05`.
6. **Encode layout.** Encode always emits `name <email> unix ±zzzz` —
   fields separated by single spaces, zone formatted as signed four
   digits derived from `When`'s own zone. Covered by `TestDetail06`.
7. **String omits time.** `Signature.String()` renders name and email
   only — no timestamp digits appear. Covered by `TestDetail07`.
8. **Type dispatch.** `DecodeObject` returns an object whose concrete
   type matches the encoded object's type; an unrecognised type produces
   an error, never a nil object with nil error. Covered by
   `TestDetail08`.
9. **Blob adoption.** `Blob.Decode` adopts the encoded object — ID, type
   and size match the source, and `Reader` streams the stored content.
   Covered by `TestDetail09`.
10. **Lazy iteration with clean stop.** `ObjectIter` decodes one element
    per `Next` call, `ForEach` propagates a callback error unchanged, and
    `storer.ErrStop` ends the walk without error. Covered by
    `TestDetail10`.
11. **Decode failure surfaces (shape).** When an element fails to decode,
    `Next` returns a non-nil error rather than silently skipping it or
    terminating cleanly. Asserted shape: a non-nil, non-EOF error.
    Covered by `TestDetail11`.
12. **GetBlob type check.** `GetBlob` on a stored object of the wrong
    type returns an error rather than decoding it as a blob. Covered by
    `TestDetail12`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | partially |
| TestDetail02 | 2 | partially |
| TestDetail03 | 3 | no — shape only |
| TestDetail04 | 4 | no — shape only |
| TestDetail05 | 5 | no — shape only |
| TestDetail06 | 6 | partially |
| TestDetail07 | 7 | partially |
| TestDetail08 | 8 | partially |
| TestDetail09 | 9 | partially |
| TestDetail10 | 10 | doc |
| TestDetail11 | 11 | no — shape only |
| TestDetail12 | 12 | partially |
