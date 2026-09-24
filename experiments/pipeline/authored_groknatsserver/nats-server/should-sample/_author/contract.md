# Contract — should-sample

Hidden suite: `tests/hidden/server/should_sample_bb_test.go`
(package `server`, in-package). One `TestDetailNN` per DETAILS.md line.
Headers reach the unit through the client's parsed-state header field; the
header-name constants already visible in the package are used rather than
re-spelled literals.

| test | detail | inferable | asserts |
|---|---|---|---|
| TestDetail01 | 1 | yes | A nil latency config returns `(false, nil)` even with a would-be-sampling header present. |
| TestDetail02 | 2 | yes | `sampling < 0` returns `(false, nil)` even with a would-be-sampling header — headers are not inspected. |
| TestDetail03 | 3 | yes | `sampling >= 100` returns `(true, nil)` with no headers and with a denying header — headers are not inspected. |
| TestDetail04 | 4 | no | With `sampling=1` and no headers, the hit rate over 20k draws is ~2% — consistent with an inclusive `<=` bound on a 0–99 draw (asserted as a count window 250–800, far from the 1% an exclusive bound would give). |
| TestDetail05 | 5 | partially | With `sampling=1` plus an always-sampling header, every call in 200 tries returns true — after a random miss the header path is consulted. (The derivable part; the miss path itself is statistical and not pinned per-call.) |
| TestDetail06 | 6 | yes | `sampling == 0` never samples without headers yet does sample on an always-sampling header — the random branch is skipped. |
| TestDetail07 | 7 | yes | Nil and empty parsed headers both return `(false, nil)`. |
| TestDetail08 | 8 | no | Uber-Trace-Id: four colon fields, last field 1–2 hex digits, low bit decides — `"a:b:c:1"`, `":01"`, `":ff"` sample; `":0"`, `":2"`, three-digit, 3-field, 5-field, and non-hex forms deny. (Field-shape and low-bit rule asserted; parsing internals not.) |
| TestDetail09 | 9 | yes | A sampled Uber value is echoed back as the Uber header key, and `Uberctx-` baggage keys are propagated. |
| TestDetail10 | 10 | yes | `X-B3-Sampled: 1` samples; the returned header carries the B3 sampled and trace-id keys. |
| TestDetail11 | 11 | yes | `X-B3-Sampled: 0` denies even with a trace id present. |
| TestDetail12 | 12 | doc | A lone `X-B3-TraceId` samples and is echoed in the returned header. |
| TestDetail13 | 13 | no | A single-field `B3: 0` denies. |
| TestDetail14 | 14 | no | Multi-field B3 denies when the third `-` field is `"0"` (`x-y-0-z`) and samples on any other third field (`d`, `1`, `s`). |
| TestDetail15 | 15 | no | A sampled combined B3 value returns a header whose only key is the combined B3 key holding the original value — not split B3 fields. |
| TestDetail16 | 16 | no | `traceparent` samples iff it has four `-` fields, a two-character flags field, and flag bit 0x1: `-01` samples; `-00`, `-02`, one-char flags, three-char flags, 3-field, and 5-field forms deny. Field lengths beyond the flags shape are not validated. |
| TestDetail17 | 17 | yes | A sampled traceparent is echoed as the trace-context key and `Tracestate` is propagated. |
| TestDetail18 | 18 | yes | Ordering: a denying Uber beats sampling B3/traceparent; `X-B3-Sampled: 1` beats a combined-B3 deny and returns the split-B3 header; a combined-B3 deny beats a sampling traceparent; a combined-B3 sample beats a denying traceparent. |

Refusals/softening: line 4's `<=` bound is asserted statistically at
`sampling=1` — the only place the bound is observable — inside a wide
binomial window, never as an implementation detail. Lines 8, 13–16 are
`Inferable: no`; each is asserted as the committed field-shape/bit rule
only — no literal error text, no internal helper names.
