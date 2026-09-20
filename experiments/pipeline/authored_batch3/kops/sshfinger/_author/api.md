# Exported API — sshfinger

Package `pkg/pki` (importable as `example.internal/clustkit/pkg/pki`).

- `func ComputeAWSKeyFingerprint(publicKey string) (string, error)` — AWS's key fingerprint: RSA keys hash the DER encoding, ed25519 keys hash the SSH wire encoding.
- `func ComputeOpenSSHKeyFingerprint(publicKey string) (string, error)` — the classic OpenSSH fingerprint: md5 of the wire encoding in colon-separated hex.
- Unexported helpers: `parseSSHPublicKey`, `rsaToDER`, `colonSeparatedHex`.

Production callers: `upup/pkg/fi/cloudup/awstasks/sshkey.go`, `pkg/commands` SSH display paths.
