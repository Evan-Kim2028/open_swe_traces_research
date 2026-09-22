# Contract — exportauth

Account export/import authorization: stream and service export lookups,
public vs approved vs token-required exports, account-position wildcards,
and subject/user revocation maps. Every commitment below is covered by a
hidden test; every hidden test maps to a commitment.

## Commitments

1. **Guarded stream check.** The stream-import authorization check returns
   false when the exporting account has no stream exports, and false when
   the requested subject is not a valid subject — even when an export
   pattern would cover it. The service-import check performs no subject
   validity gate: an invalid-but-covered subject is authorized. Covered
   by `TestDetail01`.
2. **Public export.** An export registered with no account list is
   public: the exact-subject lookup authorizes any importing account.
   Covered by `TestDetail02`.
3. **Subset direction.** A requested subject is authorized by a wildcard
   export only when the request is a subset of (covered by) the export
   pattern; a request broader than the pattern is denied. Covered by
   `TestDetail03`.
4. **Authorization order.** When an account position is configured, the
   requesting account's name appearing at that position in the requested
   subject authorizes — ahead of and independent of the approved list and
   the token requirement — and a position mismatch denies. With no
   position configured, a token-required export denies a claimless import
   and authorizes with a valid activation token; undecodable or revoked
   tokens deny. With no position and no token requirement, approved-list
   membership decides. Covered by `TestDetail04`.
5. **Service export lookup.** A service-export lookup returns the exact
   subject's export, else falls back to subset matching against the
   registered (possibly wildcard) service exports; it returns nothing for
   an uncovered subject. Covered by `TestDetail05`.
6. **Revocation map.** An empty revocation map revokes nothing. An entry
   for the subject — or the all-subjects wildcard key — with a timestamp
   at or after the issuance time revokes; an older timestamp or an entry
   for an unrelated subject does not. Covered by `TestDetail06`.
7. **User revocation.** A user is revoked exactly when the account's
   user-revocation map revokes that key at the given issuance time.
   Covered by `TestDetail07`.
8. **Wrapper parity.** The locking stream- and service-import
   authorization wrappers return the same answers as their no-lock forms
   invoked under a held read lock. Covered by `TestDetail08`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | partially — asymmetry asserted behaviorally both ways |
| TestDetail02 | 2 | yes |
| TestDetail03 | 3 | partially — subset direction and coverage only; first-match iteration order not pinned |
| TestDetail04 | 4 | partially — position/token/approved precedence asserted; unreachable branch combinations not pinned |
| TestDetail05 | 5 | partially — nil vs non-nil lookup results only |
| TestDetail06 | 6 | yes |
| TestDetail07 | 7 | yes |
| TestDetail08 | 8 | doc — asserted as wrapper/no-lock equivalence |
