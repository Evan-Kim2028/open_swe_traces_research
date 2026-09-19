# Contract (L2) — evalctx

Each design language root may be registered once; a second root with the same evaluation name is rejected. Registration also records that root's package import paths. The ordered root list is a stable topological order in which a root appears before the roots that depend on it (dependencies last). If two roots depend on each other, directly or through others, ordering fails with a cycle error naming both. The current expression is the last element of the evaluation stack, or none when the stack is empty. Recording an error appends it to the context's error list, which prints as the joined error text.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestRunDSL_ReportErrorLocation` | a registered root is ordered and executed; recorded errors are visible after the run |
| `TestRunDSL_ValidationErrorLocation` | validation errors recorded on the context are returned by the run |



Reproduce with:

```
tests/test.sh
```

That script installs the hidden suite and runs it (equivalent to
`go test -count=1 -timeout 15m ./eval/`
after the suite is in the tree). Do not skip, delete, or weaken the
tests. Do not change test assertions or testdata just to make them
green.

Work in `/app`. Keep unrelated tests passing.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
