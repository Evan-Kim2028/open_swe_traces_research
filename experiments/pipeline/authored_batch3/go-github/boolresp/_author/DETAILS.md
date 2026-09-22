# DETAILS — boolresp

1. `nil` error in → `(true, nil)` out. Inferable: yes — kept in the stub and
   forced by every consumer's contract.
2. An `*ErrorResponse` carrying HTTP 404 is swallowed to `(false, nil)` — the
   error is not propagated in this one case. Inferable: partially — mapping a
   missing resource to plain false is derivable, singling out 404 over other
   statuses is arbitrary.
3. Any other error — non-404 `*ErrorResponse` or a non-API error — is returned
   unchanged as `(false, err)`. Inferable: yes — propagation is the only sane
   default once 404 is special-cased.
4. The 404 test unwraps with `errors.As`, so wrapped error responses count.
   Inferable: no — accepting wrapped errors is an arbitrary edge choice.
