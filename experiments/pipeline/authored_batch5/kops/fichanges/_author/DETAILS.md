# Details — fichanges

1. `BuildChanges` treats a nil POINTER field in `e` as "don't care" — other nil kinds in `e`
   still count as real expected values. Inferable: yes — the contract is documented on the
   function.
2. A nil `a` (interface holding nil pointer) copies ALL non-nil `e` fields and returns true.
   Inferable: yes — documented.
3. Unexported fields (`PkgPath != ""`) are skipped entirely. Inferable: yes.
4. `equalFieldValues` order of checks: invalid → map → slice → `CompareWithID` → `Resource`
   → `DeepEqual`. Two `CompareWithID` values are equal only when BOTH IDs are non-nil and
   equal; nil IDs fall through to DeepEqual. Inferable: partially — the nil-ID fallthrough is
   subtle.
5. `Resource` comparison consults `HasIsReady` on the EXPECTED side only; a not-ready
   expected resource makes the field "changed" (returns false) without comparing bytes.
   Inferable: partially.
6. Map/slice equality requires identical nil-ness and length, then elementwise recursion;
   extra keys in `e` are NOT detected (iteration is over `a`'s keys but lengths must match,
   so it's symmetric in effect). Inferable: yes.
7. `ResourcesMatch` errors inside comparison are `klog.Fatalf` — not returned.
   Inferable: partially — fatal vs propagate is a choice.
