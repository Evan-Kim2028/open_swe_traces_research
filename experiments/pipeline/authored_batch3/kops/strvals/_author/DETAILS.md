# Details — strvals

1. `a=b` sets `dest["a"]="b"`; `a.b=c` builds the nested map `{"a":{"b":"c"}}`; `,` separates pairs. Inferable: yes — the merge contract sits in the exported doc comments.
2. `key=` at end of input stores `""` and succeeds (a trailing empty value is not an error). Inferable: partially — error-vs-empty is a choice.
3. A key ending on `,` or at EOF with no `=` is an error mentioning "has no value". Inferable: partially — the error-vs-skip choice isn't implied.
4. `a[i]=v` assigns into a list at index `i`, growing it with nils to length `i+1` when too short. Inferable: partially — growth semantics aren't stated.
5. A negative index, or an index greater than `MaxIndex` (65536), is an error. Inferable: doc — `MaxIndex` is exported and documented.
6. `a.b[i].c=v` nests maps inside list elements; an existing nil slot becomes a fresh container rather than failing. Inferable: no — reuse-vs-overwrite is internal.
7. `a={x,y}` produces a `[]interface{}` of typed values; an unterminated `{` is an error mentioning `}`. Inferable: partially.
8. Scalar coercion: `true`/`false` map to bool, `null` to nil, `0` to int64(0), other non-zero-leading integers to int64, everything else stays a string; the always-string variant never coerces. Inferable: partially — the coercion set is conventional but its exact membership is a choice.
9. `\` escapes the following character verbatim in keys and values, so `a\.b` is one key segment. Inferable: partially.
10. More than `MaxNestedNameLevel` (30) dotted segments is an error. Inferable: doc — exported constant.
11. An empty key writes nothing (the set is silently skipped). Inferable: no.
12. When a pre-existing `dest` entry has the wrong shape (e.g. `a.b` where `a` already holds a scalar), the panic is recovered into an "unable to parse key" error rather than propagating. Inferable: no — recover-to-error is arbitrary.
13. After a `}` list closes, a following non-comma rune is pushed back so parsing continues into the next pair. Inferable: no — subtle scanner behavior.
