# Contract — clientclone

`Client.Clone` produces an independent client that carries over the
parent's configuration and shares its rate-limit state, while re-scoping
credentials to the clone's own origins. Every commitment below is covered
by a hidden test; every hidden test maps to a commitment.

## Commitments

1. **Uninitialized guard.** `Clone` on a client whose inner `http.Client`
   is nil returns `errUninitialized` instead of panicking. Covered by
   `TestDetail01`.
2. **Configuration carried over.** URLs, user agent, API version range,
   token, rate-check flag, and marketplace stub all carry over to the
   clone unless overridden by `opts`. Covered by `TestDetail02`.
3. **Credential re-scoping (shape).** A token-bearing clone sends its
   credentials to the clone's own origins and does not leak the token to a
   foreign origin — asserted as observable wire behavior, not the
   base-vs-wrapped transport mechanism. Covered by `TestDetail03`.
4. **Shared rate state.** The clone shares the parent's rate-limit state —
   an exhausted limit learned by the parent short-circuits the clone's
   calls before the network. Covered by `TestDetail04`.
5. **HTTP client fields copied.** `CheckRedirect`, `Jar`, and `Timeout`
   are copied to the clone's HTTP client. Covered by `TestDetail05`.
6. **Disabled rate checks carry.** When rate checks are disabled the clone
   does not attach the shared rate state — asserted via the carried flag.
   Covered by `TestDetail06`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | doc |
| TestDetail02 | 2 | yes |
| TestDetail03 | 3 | no — shape only |
| TestDetail04 | 4 | partially |
| TestDetail05 | 5 | partially |
| TestDetail06 | 6 | partially |
