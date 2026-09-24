# Details — sshfinger

1. `parseSSHPublicKey` splits on whitespace, requires at least 2 tokens, base64-decodes token[1] — the algorithm prefix token[0] is IGNORED and a trailing comment token is silently accepted. Inferable: partially — permissive parsing is a choice.
2. `colonSeparatedHex` emits lowercase hex with a colon between every byte pair (every 2 hex chars). Inferable: yes.
3. `ComputeAWSKeyFingerprint` is key-type-dependent: RSA → md5 over the PKIX DER encoding in colon-hex; ed25519 → `SHA256:`-prefixed base64 (the OpenSSH SHA256 form); ANY other algorithm is an error. Inferable: no — the per-algorithm split is an AWS quirk.
4. `ComputeOpenSSHKeyFingerprint` is md5 over the SSH wire encoding (`Marshal()`) in colon-hex for EVERY key type — no per-algorithm branch. Inferable: no — md5-of-wire vs sha256-of-wire is arbitrary.
5. `rsaToDER` reflect-converts the ssh.PublicKey to `*rsa.PublicKey` and marshals PKIX — it does not serialize the SSH wire form. Inferable: partially — the doc comment explains the reflection trick.
