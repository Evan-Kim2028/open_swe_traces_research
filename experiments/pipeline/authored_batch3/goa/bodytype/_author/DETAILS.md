# Commitments — bodytype

1. An explicit Body DSL attribute is used as-is (dup'd and renamed);
   otherwise the request body is the payload minus attributes mapped to
   headers, params, and cookies — plus security-scheme attributes that
   default to headers (basic-auth user/pass, API key, bearer, JWT,
   OAuth token). In-tree coverage: deleted tests only. Inferable:
   partially — the subtraction is documented, the security defaults are
   not.
2. When the payload is not an object: no headers/params/cookies means
   the whole payload is the body; otherwise the body is empty. In-tree
   coverage: deleted tests only. Inferable: partially.
3. Computed body types become generated user types named
   "<Endpoint>RequestBody"/"<Endpoint>Streaming Body"/
   "<Endpoint>ResponseBody" via concat's smart-casing rules
   (underscore-lower, underscore-title, camelcase). In-tree coverage:
   deleted tests only. Inferable: no — concat casing rules are
   arbitrary.
4. Response bodies name by endpoint plus status text when multiple
   responses exist; error bodies append "_<ErrorName>". Result-type
   responses keep per-view body copies and record "name:original" meta.
   In-tree coverage: deleted tests only. Inferable: no.
5. Removing a mapped attribute also removes it from the body's required
   list and from map-valued user examples; after removal an empty
   object produces Empty. In-tree coverage: deleted tests only.
   Inferable: no.
6. extendBodyAttribute merges References (inherit) then Bases (merge)
   into the body attribute and clears both so finalize doesn't re-add
   them. In-tree coverage: none. Inferable: no.
7. walk/walkrec visit every user type reachable through object members,
   array/map element types, and union values — with cycle detection —
   and RemovePkgPath deletes "struct:pkg:path" meta from each, plus
   recursing into Bases. In-tree coverage: none. Inferable: partially.
8. copyOpenAPITypeMeta copies only openapi:typename,
   openapi:additionalProperties and openapi:/swagger: extension keys.
   In-tree coverage: none. Inferable: no.
