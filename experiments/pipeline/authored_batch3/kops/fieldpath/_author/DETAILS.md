# Details — fieldpath

1. Fields join with `.`; map keys render `[key]`; array indexes render `[n]`; wildcards render `[*]`; no leading dot on the first field. Inferable: yes — round-trip rendering is the visible contract.
2. `/` is accepted as a separator just like `.` — `a/b` parses as `a.b`. Inferable: no — unusual secondary separator.
3. `a..b` collapses to `a.b` (repeated separators are skipped, never empty elements). Inferable: no.
4. `[*]` produces a WildcardIndex element, `[0]` an ArrayIndex with the number, `[key]` a MapKey with the token. Inferable: partially — the three element kinds are exported.
5. A non-integer, non-`*` bracket body, or an unclosed `[`, is a parse error. Inferable: partially.
6. `Matches` requires equal element counts and then elementwise agreement; `HasPrefixMatch` allows the receiver to be longer. Inferable: partially — which of the two is the strict one is a choice.
7. Wildcard semantics: a WildcardIndex step in the RECEIVER matches an ArrayIndex step in the argument — it does not match MapKey steps, and a wildcard in the argument never matches anything. Inferable: no — the wildcard side is counterintuitive.
8. Element equality is exact: type, token, and number must all agree (`[a]` vs `[0]` differ even in "position"). Inferable: yes.
9. `IsEmpty` is true for a path with zero elements (e.g. parsed from `""`). Inferable: yes.
10. Unknown element types in `String()` abort the process (klog.Fatalf), they don't error. Inferable: no.
