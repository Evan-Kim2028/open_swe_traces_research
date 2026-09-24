# Exported API — namedecl

```go
type NameDeclaration struct{ /* private */ }
type PackageNameKind uint8    // NameType/NameFunction/NameConstant/NameVariable
type PackageNameVisibility uint8  // ExportedName/UnexportedName
type PackageNameOrder interface { ComparePackageName(PackageNameOrder) int }

func NewExactName(kind PackageNameKind, name string) *NameDeclaration
func NewPreferredName(kind PackageNameKind, preferred string, visibility PackageNameVisibility, order PackageNameOrder) *NameDeclaration
func (d *NameDeclaration) Name() string
func (d *NameDeclaration) Kind() PackageNameKind
func (k PackageNameKind) String() string
```

Unexported: `newDependentName`, `comparePackageNames`,
`comparePackageNameOrders`, `validateNameDeclaration`,
`PackageNameVisibility.valid`, `validatePackageNameOrder`,
`isStablePackageNameOrderType`, `NameDeclaration.packagePath`,
`NameDeclaration.preferredName`, `PackageNameKind.valid`.

## Pre-existing callers

Every generated package records declarations before names freeze;
`Generation.Freeze` sorts by comparePackageNames and assigns final
spellings.
