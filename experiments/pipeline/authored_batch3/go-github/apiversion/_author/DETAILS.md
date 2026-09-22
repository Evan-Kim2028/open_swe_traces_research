# DETAILS — apiversion

1. A request carrying no version header proceeds unchecked — the guard only
   applies when the header is set. Inferable: doc — the excised function's doc
   comment states an empty version returns nil.
2. A set version is accepted iff it falls inside the client's
   [min, max] window; both bounds are inclusive. Inferable: partially — a
   range check is derivable from the doc comment, the inclusive bounds are an
   arbitrary edge.
3. The comparison is plain string ordering on the `YYYY-MM-DD` version values,
   not date parsing. Inferable: no — the lexicographic trick is an arbitrary
   implementation choice.
4. Out-of-range requests fail with the sentinel `ErrUnsupportedAPIVersion`
   (matching `errors.Is`). Inferable: yes — the exported sentinel and its doc
   comment remain in the tree.
5. The check fires inside `bareDo`, so it gates every request path that flows
   through it, not just direct `Do` calls. Inferable: yes — the call site in
   `bareDo` remains.
