# Contract — hashparse

`HashAlgorithm`/`Hash` helpers in `util/pkg/hashing`. Every commitment below
is covered by a hidden test; every hidden test maps to a commitment.

## Commitments

1. **`Hash.String`** is `<algorithm>:<lowercase hex>` with a single colon;
   `Hex` is the bare lowercase hex digest. Covered by `TestDetail01`.
2. **Length enforcement.** `HashAlgorithm.FromString` requires exactly
   32/40/64 hex chars for md5/sha1/sha256 and errors on a mismatch or on
   non-hex input. Error text is an implementation detail — only the error
   is asserted. Covered by `TestDetail02`.
3. **`FromString`** prefers explicit `md5:`/`sha1:`/`sha256:` prefixes, then
   guesses the algorithm from bare hex length; unrecognized lengths error.
   Covered by `TestDetail03`.
4. **`NewHasher`** constructs the standard hash for the three known
   algorithms (verified against crypto/md5, crypto/sha1, crypto/sha256) and
   exits on an unknown algorithm — asserted as non-zero child-process exit.
   Covered by `TestDetail04`.
5. **`HashFile`** propagates not-exist errors unwrapped (`os.IsNotExist`
   holds on the returned error) and digests file contents with the chosen
   algorithm. Covered by `TestDetail05`.
6. **`MustFromString`** returns the parsed hash on success and exits on a
   parse error — asserted as non-zero child-process exit. Covered by
   `TestDetail06`.
7. **`Equal`** requires both algorithm and digest bytes to match; different
   algorithms with equal bytes are not equal. Covered by `TestDetail07`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | yes |
| TestDetail02 | 2 | partially — lengths enforced; error text not pinned |
| TestDetail03 | 3 | yes |
| TestDetail04 | 4 | partially — exit asserted via child process, not the mechanism |
| TestDetail05 | 5 | partially — unwrap asserted via os.IsNotExist |
| TestDetail06 | 6 | partially — exit asserted via child process |
| TestDetail07 | 7 | yes |
