# Closure — secschemes

`expr/security.go` — security scheme/requirement naming, copying and
validation.

Symbols stubbed: `(*SecurityExpr).EvalName`, `DupRequirement`, `DupScheme`,
`(*SchemeExpr).AuthoredScheme`, `HasNoSecurity`,
`EffectiveSecurityRequirements`, `(*SchemeExpr).Type`, `(*SchemeExpr).EvalName`,
`(*SchemeExpr).Hash`, `(*SchemeExpr).Validate`, `(*FlowExpr).EvalName`,
`(*FlowExpr).Validate`, `(*FlowExpr).Type`, `SchemeKind.String`.

Tests removed: `expr/security_test.go` deleted (pinned every name mapping and
the validate cases); `TestDupSchemeKeepsAuthoredScheme` trimmed from
`expr/dup_test.go`; `TestMethodExprFinalizePreservesNoSecurityMarker` trimmed
from `expr/method_test.go`.
