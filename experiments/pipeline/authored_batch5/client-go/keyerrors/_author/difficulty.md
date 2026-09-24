# Difficulty — keyerrors

predicted_flip: L1
details: 6

Missed edges: dispatch ORDER (Conflict before Retryable before
AssertionFailed...); `ErrAssertionFailed` is returned unwrapped while the
others get `errors.WithStack`; the failpoint rewrite happens before any
check; `IsErr*` use `errors.As` so wrapped values still match.

Hardness driver: mostly mechanical mapping, but a wrong dispatch order or a
missing wrap is silent — tests must probe multi-field KeyErrors to catch it.
