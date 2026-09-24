# Details — reflectfmt

1. The visitor is invoked on the root value itself first, with a nil struct-field and an empty path, before any descent. Inferable: doc — "calls visitor with v and every recursive sub-value".
2. `SkipReflection` from a visitor prunes that subtree but lets siblings continue; any other error aborts the whole walk. Inferable: doc.
3. Unexported struct fields (non-empty PkgPath) are skipped entirely — not visited. Inferable: partially.
4. With `JSONNames`, a field reports its `json:` tag name — including the literal `-` of `json:"-"` — and only falls back to the Go name when the tag is absent or empty. Inferable: no — the `-` case is an ugly edge.
5. Map entries are visited with a `MapKey` element whose token is the key rendered via `%s`; slices/arrays use `ArrayIndex` with the position. Inferable: partially.
6. A nil pointer or interface is visited once (the visitor sees it) but not descended. Inferable: no.
7. `DeprecatedDoubleVisit` additionally calls the visitor once per field with the `*reflect.StructField` set, before descending into the value. Inferable: doc — the option's doc says "visit every struct field twice".
8. `IsPrimitiveValue` is true for bool/int/uint/float/complex kinds and false for string, slice, array, ptr, interface, chan, func, map, struct — strings and slices are NOT primitive here. Inferable: doc — the doc comment states it.
9. `FormatValue`: nil or nil-pointer prints `null`; ints/floats/bools print `%v`; strings print `%q` (quoted); `fmt.Stringer` uses `String()`; everything else `%#v`. Inferable: partially — quoted strings and `null` are choices.
10. `BuildTypeName` composes `*`/`[]`/`map[k]v` prefixes over `t.Name()`; unknown kinds log an error and fall back to `t.Name()`. Inferable: yes.
11. `JSONMergeStruct` merges through a JSON marshal+unmarshal — only JSON-visible fields copy — and a marshal failure aborts the process (klog.Fatalf), it does not return an error. Inferable: partially — fatal-vs-error is a choice.
12. `IsMethodNotFound` type-asserts `*MethodNotFoundError` directly — a wrapped error is NOT detected. Inferable: no — plain assertion vs errors.As is arbitrary.
