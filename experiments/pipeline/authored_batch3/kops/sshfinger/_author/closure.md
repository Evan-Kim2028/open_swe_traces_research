# Closure — sshfinger

Package: `pkg/pki` (`example.internal/clustkit/pkg/pki`).

Files: `pkg/pki/sshkey.go` (5 funcs).

Removed functions (bodies stubbed): `parseSSHPublicKey`, `colonSeparatedHex`, `ComputeAWSKeyFingerprint`, `ComputeOpenSSHKeyFingerprint`, `rsaToDER`.

Exported entry point(s): `ComputeAWSKeyFingerprint` / `ComputeOpenSSHKeyFingerprint` — consumed by the AWS keypair task and SSH credential display.
