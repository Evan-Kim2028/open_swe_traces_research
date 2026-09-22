# Commitments — goval

1. Rendering a nil attribute fails with a "render Go value: ..." error;
   rendering with an unplanned layout fails with a "linked Go type" error; any
   deeper failure is wrapped as "render Go value for <type>: <err>". In-tree
   coverage: none. Inferable: no — exact message text is a choice.
2. When the requested layout is a named Go type, the value is rendered through
   the named type's planned value layout, not the name itself. In-tree
   coverage: none. Inferable: partially — implied by "the same names and
   pointer layout as generated service types".
3. A pointer-requested non-object value produces a `var <prefix><n> <T> = <e>`
   declaration plus the expression `&<prefix><n>`; a non-pointer value produces
   no declaration. Local counter starts at 1 and the prefix defaults to
   "defaultValue" when the caller's prefix renders empty. In-tree coverage:
   none. Inferable: no — the local naming scheme is arbitrary.
4. Map entries render sorted by the key's rendered literal, not in Go map
   iteration order. In-tree coverage: none. Inferable: doc — the renderMap
   comment states "source order independent form".
5. Map keys of non-primitive design type fail. In-tree coverage: none.
   Inferable: partially.
6. Primitive literals: bools via FormatBool; signed and unsigned ints render in
   base 10 (a negative value for an unsigned design type is an error); floats
   use the shortest 'g' form and Inf/NaN are errors; strings are double-quoted
   Go literals. In-tree coverage: none. Inferable: partially — the "literal
   accepted by the native Go type" contract pins shapes, the float/unsigned
   edge choices do not.
7. Bytes design types render `[]byte("...")` from a Go string and
   `[]byte{0x.., 0x..}` (lowercase hex, no 0 padding beyond %x) from a byte
   slice. In-tree coverage: none. Inferable: no.
8. Any values: nil renders `nil`; nested slices render `[]any{...}` and nested
   maps render `map[string]any{...}` with entries sorted by key literal; an Any
   map key that is not a string is an error. In-tree coverage: none.
   Inferable: no.
9. A field whose value already carries the declared custom Go type renders
   `<pkg.Type>(<literal>)` for scalars and `<pkg.Type>{0x.., ...}` for byte
   slices; a custom type whose declared package/name disagrees with the
   value's Go type is an error. In-tree coverage: none. Inferable: no.
10. OneOf values require a resolver; the tag and value are read from the map
    keys given by the union's type-key and value-key; an unknown branch tag is
    an error; the result is `<constructor>(<rendered branch value>)`. In-tree
    coverage: none. Inferable: partially.
11. Object field lookup reads map keys directly but resolves struct fields via
    the Goified (exported, struct:field:name-aware) field name. In-tree
    coverage: none. Inferable: partially.
