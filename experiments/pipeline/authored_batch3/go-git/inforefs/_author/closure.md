# Closure — inforefs

Package: `plumbing/protocol/packp`. File: `inforefs.go`.

Removed (3 functions stubbed): `InfoRefs.Decode`, `usableInfoRefsName`, `InfoRefs.Encode`.

Kept: type `InfoRefs{References []*plumbing.Reference}`, `ErrInvalidInfoRefs`, and the long doc
comment on `Decode` (part of the signature block — stays visible to the solver), shared wire
constants.

All 23 `*_test.go` files in the package deleted (`inforefs_test.go` exercises this codec).
