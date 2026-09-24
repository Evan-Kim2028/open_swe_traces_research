# Commitments — validcode

1. validateAttribute validates non-user-type attributes inline; alias
   user types validate against the underlying base type preserving
   pointer semantics; other user types emit a call to the scope's
   ValidatorCall guarded by nil when the resolver says nil is not
   accepted. In-tree coverage: deleted tests only. Inferable:
   partially.
2. validationAttributeNeedsNilGuard is false for arrays and maps;
   unions use IsUnionPointer under sum-type scopes or !required
   otherwise; everything else needs a guard when the field is a pointer
   or is optional without a usable default. In-tree coverage: deleted
   tests only. Inferable: no.
3. validationCode renders a block per constraint present — enum values,
   format (skipped when struct:field:type overrides a string), pattern,
   exclusive and inclusive min/max, min/max length, and required fields
   — joining them with newlines; targetVal is "*target" for pointer
   primitives and "Type(target)" for aliases. In-tree coverage: deleted
   tests only. Inferable: no — per-constraint template set is detail.
4. renderValidationPath quotes literal paths and concatenates variable
   roots with quoted suffixes. In-tree coverage: none. Inferable:
   partially.
5. hasValidations delegates to NeedsValidation under the context's
   layout policy. In-tree coverage: none. Inferable: partially.
6. toSlice renders []any{...} with %#v elements; oneof joins
   "target == v" with " || ". In-tree coverage: none. Inferable: yes.
7. constant maps each supported format name to its apikit.Format*
   constant (date, date-time, uuid, email, hostname, ipv4, ipv6, ip,
   uri, mac, cidr, regexp, json, rfc1123) and panics on unknown.
   In-tree coverage: deleted tests only. Inferable: partially — format
   list documented, constant names are not.
8. protobufUnionPayloadRequiresPresence is true when the union branch's
   unaliased type is not a plain primitive — bytes and any count as
   nilable. In-tree coverage: deleted tests only. Inferable: no.
