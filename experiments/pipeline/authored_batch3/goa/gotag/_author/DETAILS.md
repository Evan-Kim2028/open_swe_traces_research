# Commitments — gotag

1. GoNativeTypeName maps every primitive kind to its Go builtin name
   (bool, int, int32, int64, uint, uint32, uint64, float32, float64,
   string, []byte, any) and panics on non-primitives. In-tree coverage:
   deleted tests only. Inferable: yes — mechanical mapping.
2. IsNilable is true for objects, arrays, maps, and (after unaliasing)
   bytes and any. In-tree coverage: deleted tests only. Inferable:
   partially — the unalias step and bytes/any inclusion are the detail.
3. arrayElementIsPointer is true only when validation is enabled, the
   array marks NonNullableElems, and the element type is a non-nilable
   primitive. In-tree coverage: deleted tests only. Inferable: no.
4. goFieldIsPointer is true for object fields, primitive-pointer fields
   per the required/default rules, or when pointer is requested on a
   primitive that is not Any or Bytes; unions stay values. In-tree
   coverage: deleted tests only. Inferable: no.
5. AttributeTags emits ` name:"v1,v2"` entries for every "struct:tag:*"
   meta key sorted by name, skipping "struct:tag:json:name", wrapped in
   a leading space and backticks. In-tree coverage: deleted tests only.
   Inferable: no — sort order and the skipped key are details.
6. AttributeTagsWithName merges "struct:tag:json:name" into a computed
   json tag (appending ",omitempty" when the field is not required by
   its parent) unless a full "struct:tag:json" override is present;
   other struct:tag keys are emitted sorted. In-tree coverage: deleted
   tests only. Inferable: no — the merge/override precedence is the
   whole point.
