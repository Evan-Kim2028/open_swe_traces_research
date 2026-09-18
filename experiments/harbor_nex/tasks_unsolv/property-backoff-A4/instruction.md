# Missing behavior

Retry waits must follow truncated exponential backoff. With no jitter, the
wait before attempt `n` (counting from 0) is `min(cap, base * 2^n)`
milliseconds. The sequence is non-decreasing and, once it hits `cap`,
stays at `cap`. Random jitter policies (full, equal, decorr) draw from
that exponential envelope; they must not ignore `n` and return a constant.

Worked examples (no jitter):

1. base=2, cap=500, n=0 → 2
2. base=2, cap=500, n=1 → 4
3. base=2, cap=500, n=8 → 500 (2×2^8 = 512, truncated to cap)

A special case that only returns those three answers is wrong: the same
rule must hold for arbitrary base, cap, and n, including n large enough
that `base * 2^n` no longer fits in a 64-bit integer (then the wait is
`cap`). Returning `base` on every attempt is wrong: expected 4 after the second
wait with base 2, actual 2.

Reproduce with:

```
go test -count=1 -timeout 15m ./internal/client/retry/
```

Implement the missing behavior so the hidden tests pass. Do not skip,
delete, or weaken the tests. Do not change test assertions or testdata
just to make them green.

Work in `/app`. Keep unrelated tests passing. The fix must not break
existing behavior.

## Hidden unit tests (names only)

The verifier copies these tests into the tree and runs them.
Do not skip, delete, or weaken them.

- `TestBackoffExponentialProperty`: 10k seeded random expo(base,cap,n) cases plus cap/overflow/monotonic adversarial edges.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
