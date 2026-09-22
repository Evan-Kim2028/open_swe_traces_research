# DETAILS — newformreq

1. A `urlStr` that resolves to a host outside the configured origins is
   rejected (`ErrUntrustedDestination`) before the form body is posted.
   Inferable: doc — the destination rule is documented on the error.
2. A `urlStr` that *is* a configured destination is accepted.
   Inferable: yes.
3. `X-Github-Api-Version` is set to the client's default API version on
   every form request. Inferable: doc — same header contract as
   NewRequest.
4. `User-Agent` is omitted entirely when the client's user-agent is
   empty. Inferable: partially — same omit-if-empty convention as
   NewRequest.
5. `Content-Type` is the multipart boundary produced by the form buffer.
   Inferable: yes.
