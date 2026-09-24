# Commitments — protoids

1. Characters that protobuf identifiers cannot contain are replaced by
   underscores before case conversion; letters, digits and underscores pass
   through. In-tree coverage: `TestProtobufNames`, `TestProtobufFieldNames`
   (removed). Inferable: partially — "makes the first character legal for
   protobuf source" implies sanitization; the replacement character is a
   choice.
2. Everything from the first `:` onward is dropped (service-version suffixes
   never reach the emitted identifier). In-tree coverage:
   `TestProtobufNames` (removed). Inferable: no.
3. Every maximal digit run gains a trailing underscore separator, so the
   emitted Go name matches what protoc-gen-go produces for the same proto
   name. In-tree coverage: `TestProtobufNames`, `TestProtobufFieldNames`
   (removed). Inferable: doc — the helper comment commits to matching
   protoc-gen-go.
4. `ProtobufName` keeps common acronyms uppercase and upper-cases the first
   letter; `ProtobufFieldName` produces snake_case. In-tree coverage:
   `TestProtobufNames`, `TestProtobufFieldNames` (removed). Inferable: doc —
   stated in the function comments.
5. A name that sanitizes to empty yields `Val` from `ProtobufName` and `val`
   from `ProtobufFieldName`. In-tree coverage: `TestProtobufNames`,
   `TestProtobufFieldNames` (removed). Inferable: no — the fallback spelling
   is arbitrary.
6. A sanitized name starting with a digit is prefixed with `_`. In-tree
   coverage: `TestProtobufNames`, `TestProtobufFieldNames` (removed).
   Inferable: partially — "makes the first character legal" implies a fix;
   the `_` spelling is a choice.
7. A `ProtobufFieldName` result that collides with a protobuf keyword (e.g.
   `string`, `message`, `oneof`, `rpc`) gains a trailing `_`. In-tree
   coverage: `TestProtobufFieldNames` (removed). Inferable: partially —
   reserved words must be escaped; the trailing-underscore spelling is a
   choice.
