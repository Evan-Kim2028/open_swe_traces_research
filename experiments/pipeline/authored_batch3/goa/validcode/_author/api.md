# Exported API — validcode

```go
func validateAttribute(ctx *AttributeContext, att *expr.AttributeExpr, put expr.UserType, target string, context validationPath, req, view bool, seen map[expr.UserType]*bytes.Buffer) string
```

Unexported validation rendering: `validationAttributeNeedsNilGuard`,
`validationCode`, `literalValidationPath`, `parameterValidationPath`,
`validationPath.child`, `renderValidationPath`, `hasValidations`,
`toSlice`, `oneof`, `constant`, `protobufUnionPayloadRequiresPresence`.

## Pre-existing callers

Transport encoders and user-type validators call validateAttribute for
every attribute carrying validations; the produced Go source assumes an
`err` variable exists in scope.
