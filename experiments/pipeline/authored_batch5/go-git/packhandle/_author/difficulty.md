# Difficulty — packhandle

predicted_flip: L2
details: 13

Missed edges: constructor precondition split; size cache with no failure
memoisation; refcount-per-cursor; closed-check-before-FD ordering; footer
pin vs well-formed parse; meta success-only caching; close idempotence +
error join; idle-release keeping caches; short-read EOF normalisation;
negative-seek and bad-whence split; double-close no-op; missing-source
index refusal; pool-vs-grace inversion.

Hardness driver: a lifecycle contract — every rule lives in the doc
comments, but wiring atomic flags, once-values and refcount orderings so
they actually hold under concurrency is where attempts slip.
