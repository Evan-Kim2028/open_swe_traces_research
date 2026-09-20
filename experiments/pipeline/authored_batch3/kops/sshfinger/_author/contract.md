# Contract (L2) — sshfinger

`ComputeAWSKeyFingerprint` is key-type-dependent: RSA→md5 of the PKIX DER encoding in colon-hex; ed25519→`SHA256:`+base64 of the wire encoding; other algorithms error. `ComputeOpenSSHKeyFingerprint` is md5 of the SSH wire encoding in colon-hex for all types. `parseSSHPublicKey` whitespace-splits, requires ≥2 tokens, base64-decodes token[1] (prefix ignored, trailing comment tolerated). `colonSeparatedHex` is lowercase hex with a colon between bytes. `rsaToDER` reflect-converts to `*rsa.PublicKey` and marshals PKIX.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `Test_AWSFingerprint_RsaKey1` / `RsaKeyEncrypted` | RSA → DER-md5 colon-hex |
| `Test_AWSFingerprint_Ed25519Key` | ed25519 → SHA256 base64 form |
| `Test_AWSFingerprint_TrickyWhitespace` | whitespace/comment tolerance in parsing |
| `Test_AWSFingerprint_DsaKey` | non-RSA/non-ed25519 errors |
| `Test_OpenSSHFingerprint_RsaKey1` | OpenSSH md5-of-wire colon-hex |
