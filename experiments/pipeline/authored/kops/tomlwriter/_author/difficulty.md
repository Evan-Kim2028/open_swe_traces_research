# Difficulty — tomlwriter

predicted_flip: L2
The package comment is itself a full serializer spec (order, indent, quoting, go-toml v1 scalar-walk). Ten tests encode the edges. A mid model with the contract should match bytes at L2; remaining risk is `\u` case and pre-quoted passthrough.

Hardness driver: byte-stable TOML with go-toml v1 walk quirks.
