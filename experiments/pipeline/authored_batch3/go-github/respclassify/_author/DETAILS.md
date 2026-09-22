# DETAILS — respclassify

1. A 202 Accepted yields `*AcceptedError`; 2xx yields nil. Inferable: doc —
   the doc comment lists AcceptedError among the returned types.
2. The response body is decoded into ErrorResponse up to a size cap, and
   the body is restored afterward so callers can re-read it. Inferable:
   partially — re-populating the body is a documented quirk.
3. 401 with an OTP-required header yields `*TwoFactorAuthError` (a cast of
   the parsed ErrorResponse). Inferable: partially — the header constant
   survives, the trigger condition is arbitrary.
4. 403 or 429 with rate-limit-remaining 0 yields `*RateLimitError`
   carrying the parsed rate and message. Inferable: doc — the doc comment
   names the type; the remaining==0 trigger is a server convention.
5. 403 or 429 whose documentation_url ends in the abuse/secondary-limit
   fragment yields `*AbuseRateLimitError`, with RetryAfter filled from
   parseSecondaryRate when derivable. Inferable: partially.
6. 301/302/303/307/308 statuses yield `*RedirectionError` carrying the
   status code and parsed Location header. Inferable: partially — the
   status set is arbitrary.
7. `parseSecondaryRate` prefers the Retry-After header (seconds), else
   falls back to the rate-reset epoch header as a duration-until. Inferable:
   doc — both headers are described in comments.
8. Anything else yields the plain `*ErrorResponse`. Inferable: yes.
