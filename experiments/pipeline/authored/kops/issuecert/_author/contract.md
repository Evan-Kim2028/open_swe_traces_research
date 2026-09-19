# Contract (L2) — issuecert

Issuing a certificate expands well-known type aliases into comma-separated usage tokens: `ca` → CA + CRLSign + CertSign (self-signed); `client` → clientAuth + digitalSignature; `clientServer` → clientAuth + serverAuth + digitalSignature + keyEncipherment; `server` → serverAuth + digitalSignature + keyEncipherment. Unknown tokens are errors. Subject is copied; each alternate name is trimmed, IPs vs DNS split, empties skipped.

Non-CA issuance loads the signer name from the keystore; missing key or missing cert is an error. If no public key is supplied and no private key is supplied, a new RSA private key is generated (2048 unless the size env var is a valid integer). Optional Validity sets NotAfter from now (UTC). Signing fills PublicKey from the private key if needed, default NotBefore = now−48h, default NotAfter = now+10y, default 128-bit random serial if unset, default digitalSignature|keyEncipherment if KeyUsage is 0, default serverAuth ExtKeyUsage for non-CA if ExtKeyUsage is nil. Self-signed certs use the new key as parent; otherwise the CA cert/key. The returned CA pointer is the issued cert for self-signed, else the keystore CA.

PEM parse of private keys and certificates must round-trip the usual PKCS8/PKIX blocks used in tests.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestIssueCert` (ca) | self-signed CA usages and issuer==subject |
| `TestIssueCert` (client) | client usages; supplied serial kept |
| `TestIssueCert` (clientOneYear) | explicit validity window |
| `TestIssueCert` (clientServer) | DNS + IP SANs; both client and server EKU |
| `TestIssueCert` (server) | server EKU; generated vs supplied private key |
| `TestGenerateCertificate` | PEM private key parse + signed cert from existing key |
| `TestCertificateRoundTrip` | certificate PEM round-trip |
| `TestPrivateKeyRoundTripRSA` | private key PEM round-trip |
