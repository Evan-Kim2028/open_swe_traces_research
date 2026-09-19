# Difficulty — issuecert

predicted_flip: L3
Crypto issuance is a branch lattice (type aliases, self-signed vs keystore, SAN classification, default serial/validity/usages) but every branch is named in `TestIssueCert` subtests. A mid model with the contract still fumbles NotBefore skew or default ExtKeyUsage; L3 test names are enough.

Hardness driver: type-alias issuance + SAN/IP split + default validity/serial.
