# DETAILS — dounit

1. A response body that is empty or contains only whitespace produces no
   decode error for a non-nil, non-`io.Writer` target — `Do` returns
   `err == nil`. Inferable: partially — tolerating empty bodies is
   derivable (endpoints legitimately return none); that whitespace-only
   is also tolerated is an arbitrary tolerance the code happens to make.
2. A non-empty body that fails JSON decode still surfaces the decode
   error. Inferable: yes.
3. A body that cannot be *read* (transport error mid-body) surfaces the
   read error. Inferable: yes.
4. `io.Writer` targets receive the raw body bytes with no JSON decode.
   Inferable: doc — documented on the method.
5. `v == nil` performs no decode at all. Inferable: doc.
