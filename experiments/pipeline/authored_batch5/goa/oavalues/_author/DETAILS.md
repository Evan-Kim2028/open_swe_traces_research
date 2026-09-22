# Commitments — oavalues

1. `With*` return new `Values`; the receiver's maps are never mutated (copy-on-write). In-tree coverage: `TestValues`, `TestValuesOwnCompleteExampleLists` (trimmed). Inferable: doc — Values' comment states "return a new independent Values".
2. `Examples` looks up the authored attribute first, then falls back to the user type's authored attribute, else materializes the fallback list. In-tree coverage: `TestValuesUseAuthoredAttributeForCopies` (trimmed). Inferable: partially — the two-level key choice is a design detail.
3. Materialized examples are fresh copies: stored JSON values are deep-duplicated and descriptions come from `Description` applied by stored-source identity — independent of read order. In-tree coverage: `TestValuesApplyExampleDescriptionsRegardlessOfCallOrder` (trimmed). Inferable: partially.
4. `Example` overlays stored+authored user examples on a copy of the attribute, then delegates to its generator (suppressing generators return nil). In-tree coverage: `TestInitExamplesUsesReplacementDescription` (trimmed). Inferable: partially.
