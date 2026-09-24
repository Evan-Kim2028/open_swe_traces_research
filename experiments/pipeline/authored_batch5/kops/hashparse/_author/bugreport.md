# Bug report

Hash parsing/formatting is broken: `Hash.String` no longer emits `alg:hex`, `FromString` can't
recognise `sha256:`-prefixed or bare hex digests (or accepts wrong lengths), `NewHasher` doesn't
map the known algorithm names, `HashFile` wraps not-found errors it should pass through, and
`Equal` ignores the algorithm.

Expected: `String` is `algorithm:hex`; `FromString` accepts `alg:` prefixes or infers md5/sha1/
sha256 from 32/40/64-char hex; per-algorithm length is enforced; file-not-found propagates
unwrapped; equality covers algorithm and bytes.

Reproduce with:

```
go test -count=1 ./util/pkg/hashing/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the
internet; work only from the repository.
