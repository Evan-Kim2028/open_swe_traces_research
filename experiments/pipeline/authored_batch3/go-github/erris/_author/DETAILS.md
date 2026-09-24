# DETAILS — erris

1. Each `Is` first runs `errors.As(target, &v)` into its own concrete type;
   a type mismatch returns false. Inferable: yes — standard errors.Is
   plumbing, forced by the signature.
2. `compareHTTPResponse` treats nil==nil as equal, nil vs non-nil as
   unequal, and otherwise compares only StatusCode. Inferable: doc — the
   doc comment above it states "only StatusCode is checked"; nil rules are
   forced by Go semantics.
3. `ErrorResponse.Is` compares Message, DocumentationURL, response
   StatusCode, the full Errors slice element-wise, and Block (Reason plus
   nil-safe CreatedAt). Inferable: partially — the field set is forced by
   the struct, the choice of which fields count is arbitrary.
4. `RateLimitError.Is` compares Rate (struct equality), Message, and
   response StatusCode. Inferable: partially.
5. `AcceptedError.Is` compares the Raw byte payload only. Inferable: no —
   choosing Raw as the sole identity is arbitrary.
6. `AbuseRateLimitError.Is` compares Message, response StatusCode, and
   RetryAfter via nil-safe duration equality. Inferable: partially.
7. `RedirectionError.Is` compares StatusCode and Location — either both
   nil/same pointer, or both non-nil with equal String() forms. Inferable:
   partially — URL-by-string comparison is arbitrary.
8. `equalDurationPtr` treats nil==nil equal, nil vs non-nil unequal, else
   compares pointed-to values. Inferable: yes — nil-safe pointer equality
   is forced.
