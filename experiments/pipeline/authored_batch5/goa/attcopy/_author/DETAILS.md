# Commitments — attcopy

1. Copying an attribute produces a detached `AttributeExpr`: mutating the
   copy's description, validation, metadata, default value, or user examples
   never mutates the input. In-tree coverage:
   `TestAttributeGraphCopierDetachesMutableValues` (removed). Inferable: yes —
   "deep copy" and the file comment's "private expression snapshot" commit.
2. A node referenced twice in one graph is copied once; the result preserves
   sharing, including recursive links. In-tree coverage:
   `TestAttributeGraphCopierPreservesCycles`,
   `TestAttributeGraphCopierKeepsDistinctCopiesOfOneUserType` (removed).
   Inferable: doc — "Shared and recursive nodes remain shared" and "without
   merging distinct user types or breaking recursive links".
3. Recursive type graphs terminate: a user type reachable from itself is
   installed in the copy before its children are copied. In-tree coverage:
   `TestAttributeGraphCopierPreservesCycles` (removed). Inferable: yes.
4. `Original` maps each copied node back to the exact input node; nodes this
   copier did not create map to themselves. In-tree coverage:
   `TestAttributeGraphCopierMapsCopiesToOriginals` (removed). Inferable: doc.
5. Calling `Copy` on a node this copier already produced returns that node
   unchanged — copies are never re-copied. In-tree coverage: none. Inferable:
   doc — the `Copy` comment commits to it.
6. Copies retain the authored declaration and finalized state of every
   attribute. In-tree coverage: none. Inferable: doc.
7. Result types copy their views, and each copied view's `Parent` points at
   the copied result type, not the input. In-tree coverage:
   `TestAttributeGraphCopierDetachesResultTypeViews` (removed). Inferable:
   partially — view detachment is implied; the parent rewiring detail is not
   spelled out.
8. Mutable `any` payloads (slices, maps, pointers, interfaces, structs)
   inside defaults, validations, and examples are deep-copied while keeping
   their concrete Go type. In-tree coverage:
   `TestAttributeGraphCopierDetachesMutableValues`,
   `TestAttributeGraphCopierCopiesTypedValues` (removed). Inferable:
   partially — "without changing their concrete Go type" is doc; the
   reflection-level coverage is not.
9. A mutable value that forms a cycle inside itself is rejected with a
   panic; a struct with unexported fields holding mutable data is rejected
   the same way. In-tree coverage:
   `TestAttributeGraphCopierRejectsCyclicMutableValue`,
   `TestAttributeGraphCopierRejectsUnsupportedMutableValues` (removed).
   Inferable: partially — rejection is required (fail-fast norm); panic and
   message text are choices.
10. `Copy(nil)` returns nil; `Empty` and primitive types are returned
    unchanged rather than copied. In-tree coverage: none. Inferable:
    partially.
