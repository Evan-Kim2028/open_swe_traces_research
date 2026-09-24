# Commitments — errcontract

1. Two error attributes are equivalent when their value contracts match:
   descriptions, docs, and examples are documentation and never affect the
   verdict; types, validations, defaults, and metadata do. In-tree coverage:
   `TestEquivalentErrorAttributes*` (removed). Inferable: doc — stated in
   `equivalentErrorAttributes`' comment.
2. Equivalence is computed on the effective attribute — References and Bases
   are materialized by `Finalize` before comparing — and the evaluated design
   is never mutated. In-tree coverage:
   `TestEquivalentErrorAttributesMaterializeBases`,
   `TestEquivalentErrorAttributesMaterializeReferences` (removed).
   Inferable: doc — `effectiveErrorAttribute`'s comment commits to detached
   copies.
3. The effective copy's mutable contract values (validation slices, scalar
   pointers, defaults, enum values, examples) share no storage with the
   source. In-tree coverage:
   `TestEffectiveErrorAttributeSharesNoMutableContractValues` (removed).
   Inferable: partially — detachment is required for "without mutating
   evaluated design"; the reflection-level detail is not spelled out.
4. Nodes revisited through recursive user types are compared once, so
   self-recursive error types terminate. In-tree coverage:
   `TestEffectiveErrorAttributeReconnectsCopiesByOrigin` (removed).
   Inferable: yes — termination is mandatory.
5. Required-field order and enum order do not affect equivalence; the
   unordered content does. In-tree coverage:
   `TestEquivalentErrorAttributesIgnoreRequiredOrder` (removed). Inferable:
   doc — "validation order does not affect runtime behavior".
6. A union branch contributes its effective envelope keys so two unions that
   serialize differently are not equivalent. In-tree coverage:
   `TestEquivalentErrorAttributesUseEffectiveUnionKeys` (removed).
   Inferable: partially.
7. Metadata compares by key with ordered values; a nil map and an empty map,
   and nil and empty value slices, count as the same absent content.
   In-tree coverage: none. Inferable: doc — stated in the
   `equivalentErrorMetadata` comment.
8. `differingErrorQualifierSettings` returns the names of the error
   qualifiers whose presence differs between the two effective attributes.
   In-tree coverage: inherited-error mapping tests (removed). Inferable:
   partially — which qualifiers count and their names are choices.
9. A nil attribute equals only nil; an attribute never equals a different
   pointer unless the contracts match. In-tree coverage: none. Inferable:
   yes.
