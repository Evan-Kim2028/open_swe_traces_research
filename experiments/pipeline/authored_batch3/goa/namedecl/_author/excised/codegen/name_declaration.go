// This file records package-level Go names before source is written. Names that
// must remain exact are assigned first, then generated names receive numeric
// suffixes when needed. After that, the names cannot change.
package codegen

import (
	_ "fmt"
	_ "go/token"
	"reflect"
	_ "strings"
)

type (
	// PackageNameKind states whether a package-level name declares a type,
	// function, constant, or variable. All four kinds must have different names
	// within one Go package.
	PackageNameKind uint8

	// PackageNameVisibility states whether generated code outside the package can
	// use a preferred name.
	PackageNameVisibility uint8

	// PackageNameOrder sorts generated declarations that request the same name.
	// Implementations must be named, non-pointer values containing only values
	// that cannot change. Values of different concrete types are sorted by their
	// package and type names before this method is called.
	PackageNameOrder interface {
		// ComparePackageName compares two values of the same concrete type. It must
		// return a negative value when the receiver comes first, zero when the values
		// are equal, and a positive value when the receiver comes last. Reversing the
		// arguments must reverse the sign, and comparison must sort consistently.
		ComparePackageName(PackageNameOrder) int
	}

	// NameDeclaration records one package-level Go name. Name cannot be read
	// until Generation.Freeze chooses its final spelling among all declarations
	// in the package.
	NameDeclaration struct {
		kind       PackageNameKind
		visibility PackageNameVisibility
		preferred  string
		final      string
		owner      *GeneratedPackage
		exact      bool
		order      PackageNameOrder
		base       *NameDeclaration
		prefix     string
		suffix     string
		hashes     []Hasher
		frozen     bool
	}
)

const (
	// NameType marks a package-level type name.
	NameType PackageNameKind = iota + 1
	// NameFunction marks a package-level function name.
	NameFunction
	// NameConstant marks a package-level constant name.
	NameConstant
	// NameVariable marks a package-level variable name.
	NameVariable
)

const (
	// ExportedName requests a name that code outside the package can use.
	ExportedName PackageNameVisibility = iota + 1
	// UnexportedName requests a name that only code in the package can use.
	UnexportedName
)

// NewExactName records a package-level Go name that must not change. Adding the
// declaration to a generated package fails if name is invalid or already used.
func NewExactName(kind PackageNameKind, name string) *NameDeclaration {
	panic("excised: NewExactName")
}

// NewPreferredName records a generated name that may receive a numeric suffix
// when another declaration requests the same spelling. order decides which
// declaration keeps the unsuffixed name and must be a named, non-pointer value
// containing only values that cannot change.
func NewPreferredName(kind PackageNameKind, preferred string, visibility PackageNameVisibility, order PackageNameOrder) *NameDeclaration {
	panic("excised: NewPreferredName")
}

// Name returns the final Go name. It panics until Generation.Freeze chooses all
// declaration names because another declaration may still change this one.
func (d *NameDeclaration) Name() string {
	panic("excised: NameDeclaration.Name")
}

// Kind returns whether this name declares a type, function, constant, or
// variable.
func (d *NameDeclaration) Kind() PackageNameKind {
	panic("excised: NameDeclaration.Kind")
}

// String returns the declaration kind used in error messages.
func (k PackageNameKind) String() string {
	panic("excised: PackageNameKind.String")
}

// newDependentName records a name formed by adding prefix and suffix to base's
// final name.
func newDependentName(kind PackageNameKind, base *NameDeclaration, prefix, suffix string, order PackageNameOrder) *NameDeclaration {
	panic("excised: newDependentName")
}

// comparePackageNames sorts declarations by their order value rather than the
// order in which generators added them. Two distinct declarations must not
// compare equal.
func comparePackageNames(left, right *NameDeclaration) int {
	panic("excised: comparePackageNames")
}

// comparePackageNameOrders compares two validated order values without reading
// discovery order or pointer identity.
func comparePackageNameOrders(left, right PackageNameOrder) int {
	panic("excised: comparePackageNameOrders")
}

// validateNameDeclaration checks that declaration has a valid kind, visibility,
// and requested Go name before a generated package records it.
func validateNameDeclaration(declaration *NameDeclaration) error {
	panic("excised: validateNameDeclaration")
}

// valid reports whether v is one of the supported visibility values.
func (v PackageNameVisibility) valid() bool {
	panic("excised: PackageNameVisibility.valid")
}

// validatePackageNameOrder checks that order is a named value whose contents
// cannot change after a generated package records it.
func validatePackageNameOrder(order PackageNameOrder) error {
	panic("excised: validatePackageNameOrder")
}

// isStablePackageNameOrderType reports whether typeOf contains only values that
// cannot change after they are copied.
func isStablePackageNameOrderType(typeOf reflect.Type) bool {
	panic("excised: isStablePackageNameOrderType")
}

// packagePath returns the import path of the package that declares this name.
// It panics if no generated package has recorded the declaration.
func (d *NameDeclaration) packagePath() string {
	panic("excised: NameDeclaration.packagePath")
}

// preferredName returns the requested name. A dependent name uses base's final
// spelling once available, then adds its prefix and suffix.
func (d *NameDeclaration) preferredName() string {
	panic("excised: NameDeclaration.preferredName")
}

// valid reports whether k is one of the supported declaration kinds.
func (k PackageNameKind) valid() bool {
	panic("excised: PackageNameKind.valid")
}
