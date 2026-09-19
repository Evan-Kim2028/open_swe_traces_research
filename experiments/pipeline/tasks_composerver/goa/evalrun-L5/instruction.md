# Contract (L2) — evalrun

Evaluating a design walks every registered root. For each root, the engine first runs stored source functions, including any new expressions those functions append to the current set, and aborts that walk after 100 nested generations (too many generated expressions). After every root has been executed — and after any roots registered during that first pass, up to 100 extra root-generation rounds — the engine runs a prepare pass, then a validate pass, then a finalize pass, each visiting the root and then its expression sets. Validation failures are recorded as context errors; if any exist after execute or after validate, evaluation returns them and skips later phases. A nil source function is success. Running a source function always pushes the target onto the evaluation stack and pops it afterward, even when the function reports errors. The current expression is the stack top, or the distinguished top-level placeholder when the stack is empty. An incompatible-context report names the exported design function that was used in the wrong place. Wrong-typed extra arguments, missing arguments, and extra arguments each produce a single recorded error that names the expected shape.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestInvalidArgError` | a wrongly typed extra argument records one error naming the expected shape |
| `TestTooFewArgError` | a missing argument records one error naming the design function |
| `TestTooManyArgError` | extra arguments record one error naming the design function |
| `TestIncompatibleDSLIncludesTypeContext` | a design function used in the wrong surrounding type records an incompatible-context error |
| `TestRunDSL_ReportErrorLocation` | executing stored source records an error that evaluation then returns |
| `TestRunDSL_ValidationErrorLocation` | validation failures after execute are returned and later phases do not hide them |

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

## Hidden unit tests (names only)

The verifier copies these tests into the tree and runs them.
Do not skip, delete, or weaken them.

- `TestEvalrunInvalidArgError`, `TestEvalrunInvalidArgErrorRandom`, `TestEvalrunTooFewArgError`, `TestEvalrunTooFewArgErrorRandom`, `TestEvalrunTooManyArgError`, `TestEvalrunTooManyArgErrorRandom`, `TestEvalrunIncompatibleDSL`, `TestEvalrunIncompatibleDSLRandom`, `TestEvalrunRunDSLReportError`, `TestEvalrunRunDSLReportErrorRandom`, `TestEvalrunRunDSLValidationError`, `TestEvalrunRunDSLValidationRandom`, `TestEvalrunCurrentStack`, `TestEvalrunCurrentStackRandom`, `TestEvalrunExecuteNilSource`, `TestEvalrunExecuteSuccessRandom`, `TestEvalrunUnseenRandomProperty`: TestEvalrunInvalidArgError

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
