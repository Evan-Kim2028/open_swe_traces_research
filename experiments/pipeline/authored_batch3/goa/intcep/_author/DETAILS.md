# Commitments — intcep

1. An interceptor's attribute access paths are validated against the
   payload/result type it intercepts: every named path must resolve to
   an attribute that exists. In-tree coverage: deleted tests only.
   Inferable: partially.
2. EvalName produces the interceptor's display name including the
   defining package when present. In-tree coverage: deleted tests only.
   Inferable: partially.
3. Interceptor validation rejects access to attributes outside the
   intercepted type's shape (unknown field names, paths into
   non-objects). In-tree coverage: deleted tests only. Inferable: no.
