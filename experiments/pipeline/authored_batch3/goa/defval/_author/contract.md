# Contract (L2) — defval

Design evaluation validates every authored default value against the
attribute's design type and its validation rules before code generation.

An attribute with no authored default produces no errors. A present default is
checked, and nested attributes are walked once each — descent stops at named
types, so a shared named type's invalid default is reported once no matter how
many places use it. Errors identify the nested position: the offending field
name, element, map key or value, or OneOf branch surfaces in the message.

A nil authored value is rejected naming nil; interface and pointer values are
unwrapped to the concrete value before checking. Primitive-typed attributes
require a compatible authored Go value — with one escape: a value whose Go type
is exactly the custom field type declared through the `struct:field:type`
metadata, matched by package path plus the type's trailing name, is converted
to the primitive instead of rejected. Compatible numeric defaults must also
fit the design primitive: signed integers check signed bounds per bit-width
(plain int means the platform int width), unsigned likewise, float32 rejects
magnitudes beyond its range, and NaN and infinity are rejected on both float
widths.

Kinds must match: a union-typed default requires a map, an object accepts a
map or a struct, an array requires an array or slice, and a map requires a
map. An Any-typed default accepts only JSON-shaped Go values — booleans,
signed and unsigned integers, strings, finite floats, slices, and maps with
string keys — while nils are fine anywhere inside.

Enum membership compares numeric values after converting both sides to the
design primitive's representation type, so equal numbers of different Go
widths match; non-numeric enums compare by deep equality. Format and pattern
rules apply only when the authored value is a string — a non-string default on
a formatted or patterned attribute reports only its type mismatch. Minimum,
maximum and the exclusive bounds compare numerically; length rules count
runes for strings and elements for collections and bytes.

Object defaults report non-string map keys by their Go type, require every
required field (by its design name) to be present, reject unknown fields by
name, and match struct literals through the generated Go field name — the
declared field-name metadata, capitalized. Union defaults require the
canonical envelope: exactly the discriminator key and the value key, the
discriminator a string naming a declared branch, and the branch's own
contract applied to the enclosed value. Errors report in deterministic order:
map entries ordered by their printed key, object-field problems ordered by
field name, and invalid key types sorted.

## Coverage of hidden assertions

| hidden test | contract sentence |
|---|---|
| `TestDetail01` | absent defaults produce no errors; present defaults are checked; named types are not re-descended so a shared type's default reports once |
| `TestDetail02` | nested errors identify the position — the offending field name or map key surfaces |
| `TestDetail03` | nil authored values are rejected naming nil; pointers and interfaces unwrap to the concrete value first |
| `TestDetail04` | primitive compatibility is enforced, with the custom-field-type metadata escape that converts by package path plus trailing type name |
| `TestDetail05` | numeric defaults must fit the design primitive's bounds; non-finite floats rejected on both widths |
| `TestDetail06` | kind matching: union wants a map, object a map or struct, array a sequence, map a map |
| `TestDetail07` | Any accepts only JSON-shaped values — bools, ints, uints, strings, finite floats, slices, string-keyed maps — nils fine |
| `TestDetail08` | numeric enum membership compares after conversion to the design primitive's type; non-members error |
| `TestDetail09` | format and pattern apply only to string defaults; bounds compare numerically; length counts runes or elements |
| `TestDetail10` | object defaults: required design names must be present, unknown fields error by name, non-string keys report their type, struct literals match via generated Go field names |
| `TestDetail11` | union defaults need the canonical two-key envelope, a discriminator naming a declared branch, and the branch contract applied to the value |
| `TestDetail12` | error order is deterministic: map entries by printed key, object-field problems by name |
