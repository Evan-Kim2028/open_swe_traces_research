# Bug report

Key metadata flags panic on every predicate and every `ApplyFlagsOps` call,
so buffered writes cannot record assertion/lock state.

Expected: `ApplyFlagsOps(0, SetPresumeKeyNotExists)` yields `0x11`
(presume-KNE plus need-check-exists); `ApplyFlagsOps(0, SetAssertExist)`
yields `0x200` with `HasAssertExist()` true; applying `SetAssertUnknown`
afterwards yields `0x600` with `HasAssertExist()` false and
`HasAssertUnknown()` true. `ApplyFlagsOps(0, SetKeyLocked,
SetKeyLockedValueExists, SetNeedConstraintCheckInPrewrite)` yields `0x80a`
and `AndPersistent()` keeps all of it; re-applying `SetKeyLockedValueExists`
drops the constraint bit (`0xa`). `ApplyFlagsOps(0, SetPresumeKeyNotExists,
DelPresumeKeyNotExists)` returns `0`.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
