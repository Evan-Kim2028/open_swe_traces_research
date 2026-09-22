# DETAILS — brnotprot

1. The predicate fires only on errors carrying the exact upstream message
   "Branch not protected" (held in the `githubBranchNotProtected` const).
   Inferable: partially — the const remains visible, matching on the message
   field rather than status is arbitrary.
2. The match reads `ErrorResponse.Message` after `errors.As` unwrapping —
   wrapped errors and non-API errors never match. Inferable: partially —
   unwrapping is derivable, the exact field is not.
3. A match makes the caller return `ErrBranchNotProtected` (replacing the raw
   API error). Inferable: yes — the sentinel's use sites remain visible.
4. Any other error — including a 400/404 with a different message — is
   returned unchanged. Inferable: yes — propagation is the default.
