# Exported API — issuecert

`IssueCert(ctx, request *IssueCertRequest, keystore Keystore) (cert *Certificate, key *PrivateKey, ca *Certificate, err error)`

`GeneratePrivateKey() (*PrivateKey, error)` — RSA, size `DefaultPrivateKeySize` or `KOPS_RSA_PRIVATE_KEY_SIZE`.

`ParsePEMPrivateKey([]byte) (*PrivateKey, error)`

`ParsePEMCertificate([]byte) (*Certificate, error)`

Type strings `ca` / `client` / `clientServer` / `server` expand to key-usage tokens. AlternateNames that parse as IPs go to IPAddresses, else DNSNames. Type `ca` is self-signed (keystore unused); others require both cert and key for `request.Signer`. Default validity is ~10 years; NotBefore is 48h in the past when unset.

Callers: keystore / nodeup PKI tasks. In-tree tests: `TestIssueCert` (table), `TestGenerateCertificate`, `TestCertificateRoundTrip`, `TestPrivateKeyRoundTripRSA`.
