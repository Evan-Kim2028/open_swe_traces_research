# Contract — createfork

`RepositoriesService.CreateFork` posts a fork request; a 202 returns
`*AcceptedError` alongside the pending repository data. Every commitment
below is covered by a hidden test; every hidden test maps to a commitment.

## Commitments

1. **202 body decoded.** On an `*AcceptedError` response the raw body is
   still decoded into the returned `*Repository` — the pending fork's
   fields are populated alongside the error. Covered by `TestDetail01`.
2. **Error type survives decode failure.** A 202 whose body does not
   decode still returns the `*AcceptedError` — the decode failure does not
   replace it. Covered by `TestDetail02`.
3. **Normal responses behave normally.** Repository populated on success;
   a non-accepted error status propagates as an ordinary error, never as
   `*AcceptedError`. Covered by `TestDetail03`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | partially |
| TestDetail02 | 2 | partially — asserted as surviving error type |
| TestDetail03 | 3 | yes |
