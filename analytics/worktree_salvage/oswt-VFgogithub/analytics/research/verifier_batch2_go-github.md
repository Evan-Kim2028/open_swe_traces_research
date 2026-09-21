# Verifier report — authored batch 2, `go-github` family

Verifier-author role. Hidden black-box suites + Harbor ladder packages for the 14 units under
`oswt-AUgogithub/experiments/pipeline/authored_batch2/go-github/*/_author/`.
Source: `github.com/google/go-github/v92` @ `d6ef767c`, base image `ladder-base:go-github` (Go 1.26.8, offline modules).

## Method

- Suite design used only `api.md`, `DETAILS.md`, and the excised tree. `bugreport.md`,
  `contract.md`, and `gold.patch` were never read.
- Each unit has one file `tests/hidden/github/<unit>_bb_test.go`, `package github_test`,
  driving only exported API. One `TestDetailNN_*` per numbered DETAILS.md line —
  86 details → 86 tests, one-to-one (verified by count).
- All suites seed `rand.New(rand.NewSource(hiddenSeed()))` from `HIDDEN_SEED`
  (default `20260919`). Audit seed `HIDDEN_SEED=20260920` passes on all 14 golds.
- Properties are seeded-random + adversarial (unknown discriminators, malformed wire
  types, boundary sizes, wrong-key/wrong-body signatures, duplicates, nulls) —
  a hardcoded example-list cheat cannot satisfy them.
- Duplicate check: `scripts/check_unit_overlap.py --extra oswt-AUgogithub` → CLEAN
  (14 new vs 0 existing; `messages.go` and `copilot.go` shared by two units each,
  disjoint symbols — INFO only). No exclusions.
- Packaging: `build_affordance_levels` → `experiments/pipeline/tasks_batch2/go-github/<unit>-L{0,2}`,
  CURSOR host allowlist, checksum-guarded hidden tests, `network_mode="no-network"`.
- B4 caught two defects during packaging (lowercase selector call `tc.check`/`tc.verify`
  on an anonymous-struct func field); renamed field to `Verify`, re-verified.

## Preflight gate (in-image; bare×2 + gold×2 + cheat×1 + seed variation)

Every packaged dir: bare must FAIL via assertion/panic (never setup/build), gold must PASS,
cheat must FAIL. All 28 dirs PASS the gate — zero exclusions.

| unit | details | tests | L0 bare | L0 gold | L0 cheat | L0 s | L2 bare | L2 gold | L2 cheat | L2 s |
|---|---|---|---|---|---|---|---|---|---|---|
| webhooksig      | 10 | 10 | fail | pass | fail | 216 | fail | pass | fail | 172 |
| eventdispatch   |  6 |  6 | fail | pass | fail | 182 | fail | pass | fail | 182 |
| auditentry      |  6 |  6 | fail | pass | fail |  72 | fail | pass | fail | 183 |
| treeentryjson   |  4 |  4 | fail | pass | fail | 214 | fail | pass | fail | 211 |
| projectsjson    | 11 | 11 | fail | pass | fail | 269 | fail | pass | fail | 282 |
| pubkeyjson      |  5 |  5 | fail | pass | fail | 194 | fail | pass | fail | 206 |
| envreview       |  7 |  7 | fail | pass | fail | 140 | fail | pass | fail | 183 |
| copilotpoly     |  8 |  8 | fail | pass | fail | 139 | fail | pass | fail | 168 |
| ndjsonmetrics   |  4 |  4 | fail | pass | fail | 268 | fail | pass | fail | 281 |
| rulesetjson     |  8 |  8 | fail | pass | fail | 207 | fail | pass | fail | 207 |
| refescape       |  4 |  4 | fail | pass | fail | 202 | fail | pass | fail | 204 |
| customprop      |  4 |  4 | fail | pass | fail | 144 | fail | pass | fail | 170 |
| teamsupdate     |  3 |  3 | fail | pass | fail | 196 | fail | pass | fail | 211 |
| netconfig       |  6 |  6 | fail | pass | fail | 280 | fail | pass | fail | 283 |

Excluded units: none.

## Cheat-patch kill evidence (which property catches the shortcut)

| unit | cheat caught by |
|---|---|
| webhooksig | D10 — HMAC over mutated bytes accepted (raw-body binding) |
| eventdispatch | D01/D05/D06 — generic map returned instead of typed struct |
| auditentry | D06 — AdditionalFields collision on declared key did not error |
| treeentryjson | D03 — delete shape dropped path/mode/type keys |
| projectsjson | D09 — base-10 integer string field_id rejected |
| pubkeyjson | D02 — numeric key_id not converted to decimal string |
| envreview | D06 — nil WaitTimer/CanAdminsBypass emitted as null, not 0/true |
| copilotpoly | D04 — Team assignee unsupported (three-way dispatch missing) |
| ndjsonmetrics | D04 — whole-body decode fails on multi-record stream |
| rulesetjson | D08 — integer-string reviewer id left nil |
| refescape | D01 — whole-string escape encoded `/` separators |
| customprop | D03 — numeric value accepted instead of error |
| teamsupdate | D03 — parent keys absent instead of explicit null under flag |
| netconfig | D06 — one of the three validators missing (single-fault accepted) |

## Notes for the record

- `git apply` silently no-ops inside a git worktree for these patches; materialization
  uses `patch -p1` (in `scripts/verifier_batch2_gogithub.py`).
- `Event.Type` keys on the struct name (`"PushEvent"`), not the snake_case message
  type; `HookDelivery.Event` is the snake_case name. Suite tests both directions.
- Gold behaviors observed while calibrating (documented so the tests aren't mistaken
  for implementation-matching): `Timestamp` accepts unix seconds; `ProjectV2ViewSortBy`
  `null` direction decodes to `*string("")`; ruleset-rules marshal omits `parameters`
  for empty parameter structs; `AuditEntry` marshal collision errors only when the
  declared field would itself emit.
- Preflight runner: `scripts/vf_preflight_all.sh` (6-way parallel, resume-safe,
  per-dir logs in `outputs/preflight/`). Scratch checker: `scripts/vf_check.sh`.
