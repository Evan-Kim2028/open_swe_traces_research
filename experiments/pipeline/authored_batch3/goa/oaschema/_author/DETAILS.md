# Commitments — oaschema

1. Schema.Dup returns a fully independent copy: scalar fields copy,
   pointer bounds get fresh pointers, Required and Enum get fresh
   slices, Properties/Defs/Extensions/Links/Media/Items/AnyOf/
   ContentSchema/AdditionalProperties recurse — and example/default/
   extension VALUES are deep-copied through reflection so mutating a
   nested map in the copy never touches the original. In-tree coverage:
   deleted tests only. Inferable: partially — "independent" is
   documented, the reflection walk is not.
2. duplicateJSONValue/duplicateJSONReflectValue copy maps, slices,
   arrays, pointers, and interfaces while preserving concrete Go types.
   In-tree coverage: deleted tests only. Inferable: no.
3. Merge fills EMPTY fields of s from other (id, type, ref, title,
   description, format, pattern, pathStart, enum, media, items, anyOf,
   additionalProperties, defaultValue, readOnly), tightens numeric
   bounds (smaller minimums, larger maximums), and unions
   Properties/Defs/Links/Required without overwriting existing
   properties/defs. In-tree coverage: deleted tests only. Inferable:
   no — fill-empty vs tighten is per-field policy.
4. ToString renders string/int/float64/bool scalars and panics on other
   kinds; ToStringMap converts map[any]any and []any recursively into
   map[string]any/[]any. In-tree coverage: deleted tests only.
   Inferable: partially.
5. ProjectExample removes fields whose meta says not to generate
   (openapi:generate=false / swagger:generate=false), projects
   user/result types through their attribute, and base64-encodes byte
   values. In-tree coverage: deleted tests only. Inferable: no — the
   hidden-field meta key and bytes encoding are details.
6. exampleMap accepts map[string]any, map[any]any (keying via
   ToString), and any reflect map with string keys; exampleSlice accepts
   any reflect array/slice. In-tree coverage: deleted tests only.
   Inferable: partially.
7. MustGenerate honors both openapi:generate and swagger:generate meta
   ("false" disables). AdditionalPropertiesFromExpr reads only
   openapi:additionalProperties. In-tree coverage: deleted tests only.
   Inferable: partially.
