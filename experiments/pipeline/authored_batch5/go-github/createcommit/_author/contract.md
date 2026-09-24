# Contract — createcommit

`GitService.CreateCommit` posts a new commit, supporting stored signatures
and signer-generated signatures. Every commitment below is covered by a
hidden test; every hidden test maps to a commitment.

## Commitments

1. **Nil options legal.** `opts == nil` proceeds with default options —
   the request is built and sent. Covered by `TestDetail01`.
2. **Stored signature wins.** When `commit.Verification` is non-nil, its
   `.Signature` is copied to the request body's `signature` field and no
   configured signer is invoked — asserted via the sent body and a
   counting signer. Covered by `TestDetail02`.
3. **Signer-generated signature.** When `commit.Verification` is nil and
   `opts.Signer` is set, the body's `signature` comes from running the
   signer over the commit content — asserted via the sent body value and
   that the signer received non-empty input, not the canonicalization
   bytes. Covered by `TestDetail03`.
4. **Signer errors abort.** A signer error aborts the call before any
   request is built — an error returns and no HTTP request reaches the
   server. Covered by `TestDetail04`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | yes |
| TestDetail02 | 2 | partially |
| TestDetail03 | 3 | partially |
| TestDetail04 | 4 | yes |
