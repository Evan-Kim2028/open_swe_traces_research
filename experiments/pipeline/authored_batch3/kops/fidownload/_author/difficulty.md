# Difficulty — fidownload

predicted_flip: L3
details: 6
Commitments: hash-hit short-circuit, scheme-set dispatch into VFS vs HTTP, streaming-hash with post-write verification, temp+chmod+rename staging, hardened-client timeouts with non-2xx rejection, and cancel-on-close ordering.

Hardness driver: the hash check happens twice (skip-if-match before, verify after) and the VFS/HTTP split is a scheme allowlist — a uniform fetch misses both; offline tests reach this through httptest servers and memfs/file URLs.
