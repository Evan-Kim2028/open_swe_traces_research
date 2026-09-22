# Closure — hashparse

Package: `util/pkg/hashing` (`example.internal/clustkit/util/pkg/hashing`).

Files: `util/pkg/hashing/hash.go` (10 funcs).

Removed functions (bodies stubbed): `Hash.String`, `Hash.Hex`, `HashAlgorithm.NewHasher`,
`HashAlgorithm.FromString`, `MustFromString`, `FromString`, `HashAlgorithm.Hash`,
`HashAlgorithm.HashFile`, `copyToHasher`, `Hash.Equal`.

Exported entry point(s): `FromString`/`MustFromString`/`HashAlgorithm.HashFile` — used by the
asset store and file-integrity checks throughout `fi`.

Test files removed in excision: `hash_test.go`.
