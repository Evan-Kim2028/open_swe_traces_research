# DETAILS — createfork

1. On an `*AcceptedError` response the raw body is decoded into the
   returned `*Repository`, so the deferred fork's fields are populated
   alongside the error. Inferable: partially — returning both is
   derivable from the API's deferred-create contract; decoding into the
   same `fork` object is an implementation detail.
2. A `202` whose body does not decode still returns the
   `*AcceptedError` (the decode failure does not replace it).
   Inferable: partially — error precedence is arbitrary; assert the
   error type survives.
3. A non-accepted response behaves normally: repository populated on
   success, error propagated on failure. Inferable: yes.
