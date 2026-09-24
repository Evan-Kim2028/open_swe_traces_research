# Closure — gitrefs

Package: github (root). File: github/git_refs.go.
Removed bodies: CreateRef, UpdateRef, DeleteRef — stubbed to variants
that drop input validation and ref normalization: CreateRef no longer
requires non-empty `Ref`/`SHA` or adds the `refs/` prefix; UpdateRef
drops the empty-`ref`/empty-`SHA` guards and the `refs/` trim before
`refURLEscape`; DeleteRef drops the `refs/` trim. Request construction
and Do calls keep working.
Kept: ListMatchingRefs, refURLEscape, Reference/CreateRef/UpdateRef
types.
Tests removed: 4 funcs in github/git_refs_test.go
(TestGitService_CreateRef, _UpdateRef, _DeleteRef,
_UpdateRef_pathEscape).
