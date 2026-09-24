# Contract (L2) — resultview

Result types carry a canonical media-type identifier and a set of named
views; projecting a result type under a view produces a new result type.

`CanonicalIdentifier` normalizes an identifier by dropping a structured
`+suffix` media-type extension while preserving the base type and every
other parameter; input that does not parse as a media type is returned
unchanged.

`IsErrorResult` reports whether a user type's declaration origin is the
built-in error result type, so generator-produced copies of that type still
match; unrelated user types and primitives do not.

Every user type tracks its declaration origin. The origin of a fresh type
is itself; a duplicated type keeps pointing at the declaration it was
copied from; renaming a type clears that pointer, so the renamed type
begins a new origin. `Dup` deep-copies the attribute part, keeps the
identifier and the declared views, and records the copied-from type as the
origin.

Finalizing a result type applies an explicitly selected view first, then
guarantees a default view exists, finalizes the underlying user type, and
repeats the same treatment for every distinct nested result type reachable
through its attributes, deduplicated by identifier. A synthesized default
view is built from a deep copy of the type's attribute — for collection
result types it uses the element's object shape — and its parent is the
result type itself. An explicit view selection on the type projects it in
place, dropping fields outside the view; a bad selection panics because
views were already validated.

Projection is memoized per (type, view): two references to the same type
under the same view yield one shared projection, and a type whose
identifier already carries the requested view parameter is returned
unchanged. A single-type projection registers itself before descending
into fields, so recursive type graphs terminate — a recursive reference
resolves to the in-flight projection — while a projected collection
registers only after running its stored DSL.

A projected single result type requires the requested view to exist. Its
validations are duplicated with the required list narrowed to the fields
the view keeps; its description gains a suffix built from the view name;
its type name gains the title-cased view name unless the view is the
default; the projection defines only a default view. A projected
collection keeps an array shape over the projected element type and its
name keeps the collection suffix. When the source carries no reusable
generated-example identity, the projection's stable UID is its projected
identifier.

Field projection always duplicates the field's attribute, so per-field
metadata never leaks between fields that share a type. A nested result
type's view comes from a view selection on the attribute itself, else the
field's metadata, else the default view; object children project only the
fields the view keeps, and array elements project through. The projected
identifier sets or overwrites the view media-type parameter while
preserving the base type and unrelated parameters.

A view's evaluation name names the view — or reports it as unnamed — and
names its parent when it has one.

## Coverage of hidden assertions

| hidden test | contract sentence |
|---|---|
| `TestDetail01` | canonical identifier drops the `+suffix` extension, preserves base and other parameters, returns unparseable input unchanged |
| `TestDetail02` | a user type whose origin is the built-in error result type matches, including generator copies; other types and primitives do not |
| `TestDetail03` | origin is self when unset; a duplicate carries the copied-from origin; renaming clears it |
| `TestDetail04` | `Dup` deep-copies the attribute, keeps identifier and views, records the source as origin |
| `TestDetail05` | finalize applies the explicit view, ensures a default view, finalizes the user type, and treats each distinct nested result type the same way |
| `TestDetail06` | a default view is built from a deep copy of the attribute — element object shape for collections — with the result type as parent |
| `TestDetail07` | an explicit view meta projects in place, dropping fields outside the view; a bad selection panics |
| `TestDetail08` | projection memoizes per (type, view); an identifier already carrying the view parameter returns the same type |
| `TestDetail09` | unknown view errors; required narrows to view fields; description gains a view-derived suffix; type name gains the title-cased view name unless default; only a default view remains |
| `TestDetail10` | a projection registers before recursing, so recursive references terminate and resolve to the in-flight projection |
| `TestDetail11` | a projected collection stays an array over the projected element and keeps the collection suffix in its name |
| `TestDetail12` | without a reusable generated-example identity, the projection's UID is its projected identifier |
| `TestDetail13` | field attributes are always duplicated; a nested result type's view comes from the attribute's view selection, else field metadata, else default |
| `TestDetail14` | the projected identifier sets or overwrites the view parameter, preserving base and other parameters |
| `TestDetail15` | a view's eval name names the view or reports it unnamed, and names its parent when parented |
