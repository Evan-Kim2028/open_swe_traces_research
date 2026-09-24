# Contract — jwt-path-for-key

Hidden suite: `tests/hidden/server/jwt_path_for_key_bb_test.go`
(package `server`, in-package). One `TestDetailNN` per DETAILS.md line.
Keys are real freshly generated account public keys; the invalid-key case
uses a non-key string plus a real key whose last character is swapped
(checksum-broken, premise-checked with the public validation helper).

| test | detail | inferable | asserts |
|---|---|---|---|
| TestDetail01 | 1 | yes | Keys of length 0 and 1 produce an empty path. |
| TestDetail02 | 2 | yes | Strings that fail public-key validation — a non-key string and a checksum-broken key — produce an empty path. |
| TestDetail03 | 3 | yes | The base name of the result is the public key plus the committed file extension. |
| TestDetail04 | 4 | yes | With sharding off, the result equals joining the store directory with the file name — no intermediate component. |
| TestDetail05 | 5 | no | With sharding on, the result joins the directory, a short fragment of the key, and the file name — asserted as: the middle component exists, is 2 characters, and is a substring of the key. The committed "final two characters" is additionally asserted as the join consequence; the fragment's identity is the shape. |
| TestDetail06 | 6 | no | The shard component equals the key's last two characters and differs from the key's first two (skipped if a generated key coincidentally shares them) — suffix, not prefix, not hash. |
| TestDetail07 | 7 | yes | A valid key yields a non-empty path; the only failure result is the empty string (no error return exists). |

Refusals/softening: lines 5–6 are `Inferable: no` — the shard directory is
asserted as a 2-character substring of the key equal to its suffix, which is
the derivable structure ("suffix, not hash, not prefix"); no other placement
convention is pinned.
