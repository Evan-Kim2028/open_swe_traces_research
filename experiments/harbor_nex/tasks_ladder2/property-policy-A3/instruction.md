# Missing behavior

Each named wait policy is a (base milliseconds, cap milliseconds, jitter)
envelope. The published table must keep those three numbers per name:

- region-miss: 2, 500, no jitter
- lock-wait: 100, 3000, equal jitter
- server-busy: 2000, 10000, equal jitter
- pd-rpc: 500, 3000, equal jitter
- disk-full: 500, 5000, no jitter
- lock-fast: distinguished name `txnLockFast`, cap 3000, equal jitter
  (its base is supplied later from caller variables)

A constructor that is given a name, base, cap, and jitter must round-trip
those fields. Server-busy is also the sleep-exclusion entry whose limit is
600000 milliseconds (ten minutes). Collapsing every envelope to 1/1/no-jitter
is wrong: expected region-miss base 2, actual 1.

Coverage the hidden checks enforce:

- Seeded draws from the full table keep name, base, cap, and jitter; busy exclusion is 600000.
- The three worked rows plus the probe constructor plus the lock-fast name.
- Policies outside those examples still match; hardcoding three rows fails.

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

- `TestBackoffPolicyTableProperty`, `TestBackoffPolicyContractExamples`, `TestBackoffPolicyUnmentionedRandom`: 10k seeded draws over the named wait-policy table plus constructor round-trip.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
