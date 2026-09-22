# DETAILS — enturls

1. `parseURL("")` is an error; parse failures are wrapped as "invalid url".
   Inferable: doc — the function's own doc comment survives.
2. `parseURL` ensures the path ends in `/` — appending one when missing.
   Inferable: doc — stated in the surviving comment.
3. `WithEnterpriseURLs` appends `api/v3/` to the base URL's path unless the
   path already ends with it or the host is `api.*` or `*.api.*`. Inferable:
   no — the exact suffix and host exemptions are arbitrary.
4. The upload URL gets `api/uploads/` appended under the same rule plus an
   `uploads.*` host exemption. Inferable: no — arbitrary literal and
   exemption list.
5. URL mutation happens inside the option closure at client-construction
   time, not per request. Inferable: yes — option plumbing remains.
6. `WithURLs` performs only the parseURL normalization — no api/v3 rewriting.
   Inferable: partially — "verbatim vs conventional" split is derivable from
   the two options' names, not forced.
