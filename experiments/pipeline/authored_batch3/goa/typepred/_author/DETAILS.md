# Commitments — typepred

1. AsObject/AsArray/AsMap/AsUnion unwrap UserTypeExpr and ResultTypeExpr
   recursively before testing the concrete type; a UserTypeExpr whose
   underlying type is an object returns that object. In-tree coverage:
   trimmed tests in types_test.go. Inferable: yes — doc comments state
   "underlying".
2. IsObject/IsArray/IsMap/IsUnion are the nil check of the matching As*
   function and therefore also unwrap user types. In-tree coverage:
   trimmed tests. Inferable: yes.
3. IsPrimitive unwraps user types: an alias user type over a primitive is
   primitive. In-tree coverage: trimmed tests. Inferable: yes.
4. IsAlias is true exactly when the type is a UserType whose underlying
   type is primitive. In-tree coverage: trimmed tests. Inferable: yes.
5. Equal compares types structurally and recursively: same kind, equal
   array elements, equal map keys and elements, objects with the same
   attribute names and equal attribute types. Two different user types
   with identical structure are Equal even though their Hash values
   differ. In-tree coverage: trimmed TestEqual in equal_test.go.
   Inferable: yes — documented on the function.
6. QualifiedTypeName produces "array<T>" and "map<K, V>" recursively;
   other kinds return the bare type name. In-tree coverage: trimmed
   tests. Inferable: doc — the doc comment shows the format.
7. toReflectType maps every primitive kind to its Go reflect.Type, maps
   objects to map[string]any, recurses through user/result types to
   their attribute type, and builds slices and maps via
   reflect.SliceOf/reflect.MapOf. A map key of object kind degrades to
   the any interface type. In-tree coverage: none directly; exercised
   via example/map code paths. Inferable: partially — the kind mapping
   is mechanical, the object-key fallback is not.
