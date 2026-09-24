# Details — jsonxform

1. `Transform` mutates the input map in place — string/slice callbacks replace values by
   writing back into the parent container. Inferable: yes.
2. Paths use `.`-joined keys with no escaping (a key containing `.` produces an ambiguous
   path); the root path is `""` and the first child is `.key` (leading dot). Inferable:
   no — leading-dot quirk is arbitrary.
3. Slice elements are visited at `path[]` (no index). Inferable: no — index-less path
   syntax is a choice.
4. `visitAny` handles `map[string]any`, `[]any`, `int64`, `float64`, `bool`, `string`, and
   nil; ANY other type (e.g. `int`, `[]string`, nested struct) errors
   `unhandled type at path`. Inferable: partially — the type whitelist is specific.
5. Object transforms see the map BEFORE children are visited; a transform that mutates the
   map affects what children see. Inferable: partially.
6. String transforms compose left-to-right (each sees the previous output); an error aborts
   the walk. Inferable: yes.
7. `SortSlice` sorts by JSON encoding — `{` sorts after `"`, numbers by their textual form;
   equal encodings preserve input order only coincidentally (`sort.Slice` is unstable).
   Inferable: partially — stability is unspecified.
8. `visitPrimitive` passes through unchanged. Inferable: yes.
