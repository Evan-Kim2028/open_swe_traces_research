# Bug report — exportauth

Account export/import authorization is missing. All check*ImportAuthorized
and check*ExportApproved entry points, the per-record `checkAuth`
decision, the wildcard service-export lookup, and the revocation
predicates panic with `excised`. Accounts cannot import streams or
services from each other; every authorization check fails closed (panic).

Reproduce:

    go test ./server/ -run 'TestImportAuthorized|TestServiceExportWithWildcards|TestAccountIsolationExportImport'

Restore the full decision chain: exact-match-then-subset-walk over the
export tables, the public/approved/accountPos/tokenReq ordering inside
checkAuth (accountPos indexes the REQUESTING subject tokens, 1-based),
wildcard service-export lookup, and the subject-or-`*` revocation
predicate with its issuedAt freshness rule.

Work only from the repository and test output. Do not use web search or
any tool that accesses the internet. This repository is fully
self-contained; do NOT fetch upstream sources. Preserve all unrelated
tests.
