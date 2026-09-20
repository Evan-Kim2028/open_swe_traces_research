# Closure — certdesc

Package: `pkg/pki` (`example.internal/clustkit/pkg/pki`).

Files: `pkg/pki/cert_utils.go` (6 funcs).

Removed functions (bodies stubbed): `PkixNameToString`, `keyUsageToString`, `parseKeyUsage`, `extKeyUsageToString`, `parseExtKeyUsage`, `BuildTypeDescription`.

Exported entry point(s): `PkixNameToString` / `BuildTypeDescription` — certificate subject rendering and cert-type classification used by `kops get` / certificate display, plus the `IssueCert` type expansion via `wellKnownCertificateTypes` (which shares the same string tables).
