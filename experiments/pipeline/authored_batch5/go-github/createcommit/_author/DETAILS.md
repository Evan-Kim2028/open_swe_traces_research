# DETAILS — createcommit

1. `opts == nil` is legal: the call proceeds with default options.
   Inferable: yes — the signature takes a pointer and documents nothing
   requiring it.
2. When `commit.Verification` is non-nil, its `.Signature` is copied to
   the request body's `Signature` field and no signer is invoked.
   Inferable: partially — that a stored signature should pass through is
   derivable; that it outranks a configured Signer is an arbitrary
   precedence.
3. When `commit.Verification` is nil and `opts.Signer` is set, the
   body's `Signature` comes from running the signer over the
   canonicalized commit fields. Inferable: partially — "Signer produces
   the signature" is derivable; the canonicalization input is not.
4. Signer errors abort the call before any request is built.
   Inferable: yes.
