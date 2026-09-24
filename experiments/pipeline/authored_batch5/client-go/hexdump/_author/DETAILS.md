# Details — hexdump

1. `Hex(msg).String()` renders a proto struct as `{Field:value ...}` with
   a single space between fields and NO field skipped except those named
   `XXX_*`; nested message fields recurse into the same brace form.
   Inferable: no — the layout is arbitrary; only its shape is asserted.
2. `[]byte` fields render as lowercase hex (`deadbeef`); the
   hex-encoding applies only when the slice ELEMENT kind is `uint8` —
   a `[][]byte` field prints `[a bc]` (inner bytes shown via `%s` of the
   outer slice), and a `[]uint64` field prints
   `[%!s(uint64=1) %!s(uint64=22)]` because the code passes `%s` on a
   non-byte slice. Inferable: no — the `%s` fallback is a latent quirk.
3. A nil pointer field prints `<nil>`; a nil MESSAGE argument also
   prints `<nil>` (the Ptr branch handles top-level nil too).
   Inferable: partially — `<nil>` is fmt's own spelling.
4. Scalar/enum fields print via `%v`, so proto enums render by name
   (`Voter`, `SI`, `NotAllowedOnFull`). Inferable: doc — follows from the
   `%v` verb on `Stringer` types.
