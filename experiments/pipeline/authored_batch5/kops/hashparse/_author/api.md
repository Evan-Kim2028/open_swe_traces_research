# Exported API — hashparse

Package `util/pkg/hashing` (importable as `example.internal/clustkit/util/pkg/hashing`).

`HashAlgorithm` (`"md5"`, `"sha1"`, `"sha256"`) and `Hash{Algorithm, HashValue}` helpers.

- `func (h *Hash) String() string` — `"<algorithm>:<hex>"`.
- `func (h *Hash) Hex() string` — lowercase hex of the digest.
- `func (ha HashAlgorithm) NewHasher() hash.Hash` — md5/sha1/sha256 constructor; exits on
  unknown algorithm.
- `func (ha HashAlgorithm) FromString(s string) (*Hash, error)` — hex-decodes `s`; requires
  exactly 32/40/64 hex chars for md5/sha1/sha256 respectively.
- `func FromString(s string) (*Hash, error)` — parses `"alg:hex"` by prefix, else guesses the
  algorithm from hex length (32 → md5, 40 → sha1, 64 → sha256); other lengths error.
- `func MustFromString(s string) *Hash` — fatal on parse error.
- `func (ha HashAlgorithm) Hash(r io.Reader) (*Hash, error)` — streams a digest.
- `func (ha HashAlgorithm) HashFile(p string) (*Hash, error)` — opens + hashes a file;
  `os.IsNotExist` propagates unwrapped.
- `func (l *Hash) Equal(r *Hash) bool` — algorithm AND bytes equal.

Example: `FromString("sha256:" + 64 hex chars)` → sha256 hash; bare 64-hex → sha256;
bare 40-hex → sha1.
