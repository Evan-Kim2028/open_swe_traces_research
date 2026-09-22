# Details — hashparse

1. `Hash.String` is `algorithm + ":" + hex` — one colon, lowercase hex. Inferable: yes —
   the `FromString` prefix parse implies the format.
2. `HashAlgorithm.FromString` enforces the exact hex length per algorithm (md5=32, sha1=40,
   sha256=64) and errors `invalid %q hash - unexpected length %d` on mismatch, before any
   hex decode. Inferable: partially — exact error text arbitrary, lengths forced.
3. `FromString` tries `md5:`/`sha1:`/`sha256:` prefixes first, then guesses by bare length;
   an unrecognized length errors `cannot determine algorithm for hash length: %d`.
   Inferable: yes — prefix-then-length is the only consistent reading.
4. `NewHasher` supports only the three algorithms and `klog.Exitf`s on anything else.
   Inferable: partially — exit vs error is a choice.
5. `HashFile` returns `os.IsNotExist` errors UNWRAPPED (other errors are wrapped with
   `error opening file %q: %v`). Inferable: partially — the unwrap special case.
6. `MustFromString` fatals on error (`FromString(%q) failed`). Inferable: partially.
7. `Equal` compares algorithm and digest bytes; different algorithms with equal bytes are
   NOT equal. Inferable: yes.
