# Closure — exportauth

Package `server`, file `server/accounts.go`.

Removed (stubbed):
- `(*Account).checkStreamImportAuthorized`,
  `checkStreamImportAuthorizedNoLock`, `checkStreamExportApproved`
- `(*Account).checkServiceImportAuthorized`,
  `checkServiceImportAuthorizedNoLock`, `checkServiceExportApproved`
- `(*Account).checkAuth`
- `(*Account).getServiceExport`, `getWildcardServiceExport`
- `isRevoked`, `(*Account).checkUserRevoked`

Retained: `exportAuth`/`streamExport`/`serviceExport`/`importMap` types,
`AddStreamExport`/`addStreamExportWithAccountPos`,
`AddServiceExport`/`addServiceExportWithResponseAndAccountPos` and other
registration paths, `checkActivation`/`isIssuerClaimTrusted` (JWT
activation), `isSubsetMatch` (sublist.go), `IsValidSubject`, `jwt.All`,
the account lock.

No import changes. Tests snipped: `TestImportAuthorized`
(server/accounts_test.go). No test files deleted; export/import
integration tests (`TestServiceExportWithWildcards`,
`TestImportExportConfigFailures`, …) still drive the closure and panic
under the bare excision.
