# DETAILS — clientclone

1. `Clone` on a client whose inner `http.Client` is nil returns
   `errUninitialized` instead of panicking. Inferable: doc — the error
   value exists for this guard.
2. All configuration fields carry over to the clone (URLs, user agent,
   API version range, token, rate flags, retry bound, marketplace stub)
   unless overridden by `opts`. Inferable: yes — "clone" semantics.
3. When the clone carries an auth token, its transport starts from the
   receiver's *unwrapped* base transport so the clone's own credential
   wrapper scopes to the clone's allowed origins; without a token the
   receiver's configured transport is reused. Inferable: no — the
   base-vs-wrapped choice is an internal fix for credential re-scoping;
   assert observable behavior (token not leaked to foreign origins,
   transport not double-wrapped).
4. The clone shares the parent's rate-limit map and secondary-limit
   reset state, guarded by the parent's rate mutex.
   Inferable: partially — shared state is derivable from "the same rate
   budget"; which structures are shared is internal.
5. `CheckRedirect`, `Jar`, and `Timeout` are copied to the clone's HTTP
   client. Inferable: partially.
6. When rate checks are disabled the clone does not attach the shared
   rate state. Inferable: partially.
