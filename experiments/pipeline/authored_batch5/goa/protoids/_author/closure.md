# Closure — protoids

`codegen/protobuf.go` — protobuf identifier serialization: authored design
name → legal protobuf message/service/method name (`ProtobufName`) and
field/oneof name (`ProtobufFieldName`), sharing `protobufIdentifier`
(character sanitization, `:` truncation, digit-run separation, empty
fallback, leading-digit fix) plus the protobuf keyword table.

Symbols stubbed: `ProtobufName`, `ProtobufFieldName`, `protobufIdentifier`.

Tests removed: `TestProtobufNames` and `TestProtobufFieldNames` trimmed from
`codegen/funcs_test.go` (pin every commitment);
`TestGeneratedPackageImportPath` trimmed from
`codegen/generator/generated_package_import_test.go` (reaches the surface
through package import naming).
