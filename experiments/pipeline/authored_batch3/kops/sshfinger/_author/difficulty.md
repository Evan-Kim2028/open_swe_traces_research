# Difficulty — sshfinger

predicted_flip: L3
details: 5
Commitments: whitespace-tolerant two-token parsing with ignored prefix, colon-hex formatting, the RSA→DER-md5 / ed25519→SHA256-base64 AWS split, uniform md5-of-wire for OpenSSH, and DER via reflection+PKIX.

Hardness driver: two plausible fingerprint recipes exist (md5-of-wire, sha256-of-wire, md5-of-DER) and each entry point picks a different combination per key type — a uniform implementation is wrong for exactly half the cases.
