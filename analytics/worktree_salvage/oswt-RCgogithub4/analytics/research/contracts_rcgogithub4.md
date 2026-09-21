# Contracts for authored_batch4/go-github (RCgogithub4)

Date: 2026-09-21. Author: reconciler pass for job RCgogithub4.

The 28 verified units under `experiments/pipeline/authored_batch4/go-github/` shipped
with `_author/gold.patch`, `_author/bugreport.md`, and a hidden suite under `tests/` but no
`_author/contract.md`, so `stage_units.py` could not stage them at any rung. This pass
copied the units in from `oswt-VFbatch4gogithub` and wrote one contract per unit in the
`authored_batch3/kops` shape: a prose paragraph of behavioural commitments plus a coverage
table pairing each hidden test with the sentence that justifies it.

## Coverage accounting

Every hidden `TestDetailNN` is paired with exactly one contract row and every row names at
least one hidden test — a strict bijection, verified by replaying the pipeline's own row
regex (`_SENTENCE_RE` in `synth/composerver_batch.py`) over each contract. "Commitments" is
the coverage-table row count; "assertions" is the number of `Test*` functions in the hidden
suite.

| unit | commitments | hidden assertions |
|---|---|---|
| `alertid` | 5 | 5 |
| `auditcoerce` | 7 | 7 |
| `auditentry` | 6 | 6 |
| `branchrules` | 6 | 6 |
| `commitraw` | 5 | 5 |
| `copilotseat` | 7 | 7 |
| `copilotspace` | 6 | 6 |
| `envdefaults` | 5 | 5 |
| `eventpayload` | 8 | 8 |
| `getcontent` | 6 | 6 |
| `ndjson` | 5 | 5 |
| `payloadbody` | 9 | 9 |
| `pkgversion` | 6 | 6 |
| `projectitem` | 6 | 6 |
| `propval` | 6 | 6 |
| `pubkey` | 6 | 6 |
| `putbuf` | 4 | 4 |
| `repocreate` | 5 | 5 |
| `reporule` | 6 | 6 |
| `reqreviewer` | 6 | 6 |
| `reviewerid` | 7 | 7 |
| `rulesetcodec` | 7 | 7 |
| `sha1` | 5 | 5 |
| `stringify` | 7 | 7 |
| `treeentry` | 5 | 5 |
| `viewsort` | 6 | 6 |
| `websig` | 7 | 7 |
| `websub` | 5 | 5 |
| **total** | **169** | **169** |

`pullraw` is excluded: its `_author/` has `gold.patch` and `cheat.patch` but no
`bugreport.md` and no `tests/` — it is not one of the verified units.

## Deliberate non-pins (shape asserted, literal withheld)

The hidden suites flag these as committed-but-unguessable choices; the contracts assert the
shape only, so a solver is not ambushed by an arbitrary literal:

- `auditcoerce` — whether empty `org`/`org_id` arrays also surface in `AdditionalFields`,
  and the nil-valued-key pruning rule, are not graded and not claimed.
- `commitraw` — the media-type version prefix is not graded; the contract asserts only the
  `diff`/`patch` family selection.
- `copilotseat`, `copilotspace`, `reqreviewer`, `propval`, `pubkey`, `reviewerid`,
  `viewsort`, `websig`, `getcontent`, `eventpayload` — error message texts are never
  pinned; the contracts commit to "returns an error" only.
- `projectitem` — for an unrecognised `content_type`, whether `Content` is left nil or an
  empty struct is not graded; the contract asserts only that no typed member materialises
  and no error occurs.
- `putbuf` — the 1 MiB bound is never stated as a literal; the contract refers to the
  solver-visible constant `maxPooledBufferCap` (named in `api.md`).
- `repocreate` — *which* preview media types is arbitrary; the contract asserts only that
  `Accept` advertises previews.
- `rulesetcodec` — element order on marshal is not graded; asserted as a set.
- `sha1` — the `If-None-Match` quoting convention is asserted only as "header carries the
  SHA" (gold emits the quoted entity-tag form).
- `treeentry` — suppression of `size`/`content`/`url` under the delete marker is
  explicitly not pinned by the suite; the contract does not demand it.
- `viewsort` — a JSON-null `direction` is not graded; only non-null wrong-typed values are.

## Assertions justified only by a committed literal (flagged)

- `payloadbody` `TestDetail06` greps the error for the substring `"exceeds maximum allowed
  size"`. This is an arbitrary literal the suite does grade, so the contract carries it as
  a quoted phrase — the only forced literal in the batch. Justified: the assertion is real
  and the phrase is also implied by the solver-visible size-cap constant.
- `stringify` `TestDetail01`/`TestDetail07` pin `<nil>` and `Stringify(123) == "123"`
  verbatim. Both are the documented output format (the bug report quotes it), so the
  contract states them.

## Notes on derivability

- `copilotspace` `TestDetail02` (org-shaped owner map → `*Organization`) is marked
  "inferable: no" in the suite, but the bug report states the `hooks_url` heuristic
  explicitly, so the contract commits to `*Organization`.
- `eventpayload` `TestDetail04` does not pin nil-`Type` with a valid `RawPayload`; the
  contract commits `empty Type → error`, which is consistent with the gold path
  (`typeToMessageMapping[""]` dispatches to the generic type and the unmarshal fails).

No hidden assertion was left without a derivable commitment. No contract row lacks a
backing test.
