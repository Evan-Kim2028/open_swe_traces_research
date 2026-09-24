# Contract (L2) — certdesc

`PkixNameToString` flattens the RDN sequence to comma-joined `key=value`, mapping 2.5.4.x OIDs to cn/serial/c/l/o/ou and falling back to numeric OID strings. `keyUsageToString`/`extKeyUsageToString` render set usages as their Go-identifier names (`KeyUsageCertSign`, `ExtKeyUsageServerAuth`), unknown ext usages as `ExtKeyUsage:<n>`. `parseKeyUsage`/`parseExtKeyUsage` exact-match back with ok-returns. `BuildTypeDescription` collects `CA`+usage names, sorts, comma-joins, then collapses via `wellKnownCertificateTypes` (e.g. `CA,KeyUsageCRLSign,KeyUsageCertSign`→`ca`) — the same comma format `IssueCert` parses back.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestIssueCert` | type expansion through the shared usage tables |
| `TestGenerateCertificate`, `TestCertificateRoundTrip` | cert generation + type description round-trip |
