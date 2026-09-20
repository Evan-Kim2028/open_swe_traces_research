# Commitments — resultview

1. The canonical identifier drops the "+suffix" media-type extension but
   preserves other parameters; an unparseable identifier returns
   unchanged. In-tree coverage: deleted tests only. Inferable: no.
2. IsErrorResult matches any user type whose ORIGIN is the built-in
   error type — including generator copies. In-tree coverage: none.
   Inferable: partially.
3. Origin returns the earliest declaration via the stored origin
   pointer (self when unset); Rename clears that pointer so the renamed
   type starts a new generated-declaration origin. In-tree coverage:
   none. Inferable: no.
4. Dup deep-copies the user-type part, keeps Identifier and Views, and
   carries the ORIGIN (not self). In-tree coverage: none. Inferable:
   partially.
5. Finalize applies the explicit view first, ensures a default view,
   finalizes the user type, then walks the attribute giving EACH
   distinct nested result type (deduped by identifier) the same
   treatment. In-tree coverage: none. Inferable: no.
6. The default view is built from a DEEP copy of the type attribute;
   array-typed result types use the element's object shape; the view's
   parent is the result type. In-tree coverage: none. Inferable: no.
7. An explicit view meta triggers an in-place projection; failure is a
   panic because the view was validated earlier. In-tree coverage:
   none. Inferable: partially.
8. Projection is memoized by (type hash, view); an identifier already
   carrying the requested view param returns the type unchanged.
   In-tree coverage: none. Inferable: no.
9. A projected single type: unknown view errors; validations are
   duplicated with Required filtered to view fields; description gains
   a " (<view> view)" suffix (defaulting to "<TypeName> result type");
   TypeName gains Title(view) unless the view is the default; the new
   type defines only a default view and inherits the view's examples.
   In-tree coverage: deleted tests only. Inferable: no.
10. The projection registers itself BEFORE recursing into fields so
    recursive references terminate; a projected collection registers
    AFTER running its stored DSL. In-tree coverage: none. Inferable:
    no.
11. Projected collections: the element type must be a result type; the
    projected name is "<ElemTypeName>Collection"; the collection's
    stored DSL executes during projection. In-tree coverage: none.
    Inferable: no.
12. Synthesized types reuse the source's generated-example identity
    when present so examples stay stable; otherwise they take the
    projected identifier as UID. In-tree coverage: deleted tests only.
    Inferable: no.
13. Field projection always duplicates the attribute — per-field meta
    never leaks across fields sharing a type. Nested result types pick
    the view from the VIEW attribute meta, else the field meta, else
    default; user types memoize by hash before recursing; object
    children project only fields present in the view; array elements
    project through. In-tree coverage: deleted tests only. Inferable:
    no.
14. The projected identifier sets/overwrites the "view" media-type
    parameter, preserving base and other params. In-tree coverage:
    none. Inferable: partially.
15. View eval name is "view \"<name>\"" or "unnamed view", plus
    " of <parent>" when parented. In-tree coverage: none. Inferable:
    partially.
