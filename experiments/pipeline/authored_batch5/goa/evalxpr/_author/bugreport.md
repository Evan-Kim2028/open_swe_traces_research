# Bug report

Running any DSL panics or mis-converts expression sets. Expected:
`ToExpressionSet` builds ordered `ExpressionSet`s from expression slices (nil
→ nil, non-slice → panic), `DSLFunc` exposes its stored DSL, `TopExpr` names
the root in errors. Got: panics whenever the engine processes a DSL set.

Reproduce with:

```
go test -count=1 ./eval/ ./expr/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.
