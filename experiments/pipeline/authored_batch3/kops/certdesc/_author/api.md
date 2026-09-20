# Exported API — certdesc

Package `pkg/pki` (importable as `example.internal/clustkit/pkg/pki`).

- `func PkixNameToString(name *pkix.Name) string` — render an X.509 subject as `cn=x,o=y` pairs.
- `func BuildTypeDescription(cert *x509.Certificate) string` — describe a cert by its usages; canonical combos collapse to short names (`ca`, `server`, `client`, `clientServer`).
- Unexported: `keyUsageToString`/`parseKeyUsage` (bitfield↔`KeyUsage*` names), `extKeyUsageToString`/`parseExtKeyUsage` (`ExtKeyUsage*` names).

Production callers: `IssueCert` type expansion (same table via `wellKnownCertificateTypes`), `pkg/commands` certificate display, `fi` keystore descriptions.
