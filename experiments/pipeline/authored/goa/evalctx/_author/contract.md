# Contract (L2) — evalctx

Each design language root may be registered once; a second root with the same evaluation name is rejected. Registration also records that root's package import paths. The ordered root list is a stable topological order in which a root appears before the roots that depend on it (dependencies last). If two roots depend on each other, directly or through others, ordering fails with a cycle error naming both. The current expression is the last element of the evaluation stack, or none when the stack is empty. Recording an error appends it to the context's error list, which prints as the joined error text.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestRunDSL_ReportErrorLocation` | a registered root is ordered and executed; recorded errors are visible after the run |
| `TestRunDSL_ValidationErrorLocation` | validation errors recorded on the context are returned by the run |
