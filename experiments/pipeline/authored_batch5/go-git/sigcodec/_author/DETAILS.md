# Details — sigcodec

1. Signature decode anchors on the LAST '<' and LAST '>' — a name or
   email containing angle brackets is still split correctly, and a
   missing or inverted bracket leaves the signature untouched, not
   zeroed. Inferable: partially — the bracket scan is implied by the
   field shapes.
2. The name is the bracket-trimmed prefix; the email is the raw interior
   — no validation, no trimming inside the brackets. Inferable:
   partially — the comments warn email "cannot be assumed well-formed".
3. Timestamp and timezone decode only when at least one separator space
   follows the '>' — a signature without time keeps When zero-valued.
   Inferable: no — the +2 offset check is arbitrary.
4. A negative timezone hour negates the minutes component too — "-0230"
   is -(2h30m), not -2h+30m. Inferable: no.
5. Encode clamps negative Unix times to zero — a pre-epoch When serializes
   as 0, never a negative stamp. Inferable: no.
6. Encode always writes `name <email> unix ±zzzz` — the timezone comes
   from When's own zone, formatted numeric. Inferable: partially — the
   layout is observable output; the clamp and zone choice are details.
7. String() drops the timestamp entirely — name and email only.
   Inferable: partially.
8. DecodeObject dispatches on the encoded object's type to the matching
   decoder — an unknown type is an error, not a nil return. Inferable:
   partially.
9. Blob.Decode adopts the encoded object — size, hash and a reader over
   its content; the stored object is what later Reader calls stream.
   Inferable: partially.
10. ObjectIter wraps an encoded-object iterator and decodes each element
    lazily on Next — ForEach stops on the callback's error and treats
    storer.ErrStop as a clean end. Inferable: doc — the iterator doc
    pattern is kept.
11. A failed Decode in the iterator surfaces the error from Next — it
    does not skip the bad object. Inferable: no.
12. GetBlob asserts the stored object is really a blob — wrong-type
    lookups error rather than silently decode. Inferable: partially.
