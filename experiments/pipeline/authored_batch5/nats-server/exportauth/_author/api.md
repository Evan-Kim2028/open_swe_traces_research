# Exported API — exportauth

Package `server` (module `example.internal/msgkit/v2`) — account
export/import authorization: the predicate chain deciding whether account
B may import subject S exported by account A, including public exports,
approved-account lists, activation-token requirements, and
account-position wildcard exports.

Surface (accounts.go):
- `(a *Account) checkStreamImportAuthorized(account, subject, imClaim)
  bool` (+ `...NoLock`) — stream side entry.
- `(a *Account) checkServiceImportAuthorized(account, subject, imClaim)
  bool` (+ `...NoLock`) — service side entry.
- `(a *Account) checkStreamExportApproved(account, subject, imClaim) bool`
  / `checkServiceExportApproved` — export-table matching.
- `(a *Account) checkAuth(ea *exportAuth, account, imClaim, tokens)
  bool` — the auth decision for one export record.
- `(a *Account) getServiceExport(subj) *serviceExport` /
  `getWildcardServiceExport(from)` — exact-then-wildcard lookup.
- `isRevoked(revocations map[string]int64, subject string, issuedAt int64)
  bool` — revocation-map predicate.
- `(a *Account) checkUserRevoked(nkey string, issuedAt int64) bool`.

Callers: `addStreamImport`/`addServiceImport` setup paths, client
publish/delivery permission checks, JWT account update re-validation.
`checkActivation`, `isIssuerClaimTrusted`, `isSubsetMatch` (sublist.go),
`AddStreamExport`/`AddServiceExport` registration, and `exportAuth`/
`streamExport`/`serviceExport` types stay visible.
