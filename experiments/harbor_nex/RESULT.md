# harbor_nex — Nex-N2.5-Pro (free) as a Harbor difficulty prober

Date: 2026-09-18. Model `openrouter/nex-agi/nex-n2.5-pro:free`, agent `terminus-2` (`reasoning_effort=max`). Tasks built by `scripts/harbor_tasks.py` from `experiments/codegraph_bugs/` patches. **Do not commit.**

**Verdict:** Nex-free **can** probe these bugs (daily 3/3 pass@1; two 1-hop client-go bugs pass), but it is a **poor production prober** on the OpenRouter free tier. The binding constraint is the **1000 req/day** `free-models-per-day-high-balance` cap (288× HTTP 429 in this job, 0× 502). Harbor retried 60 times then marked 12/18 trials `RateLimitError`. Wall cost for a successful client-go rollout is **20–42 min** (59–76 API calls). Use a paid key, or one scored attempt per day, if you want a full 2-attempt sweep.

## Invocation

Smoke (valid task tree, after rebuilding TwoSumBest so the bug is actually present):

```bash
set -a; . /home/evan/Documents/eval_tasks/.env; set +a
setsid nohup harbor run \
  --path experiments/harbor_nex/tasks/dailycodingproblem-go-twosumbest \
  --agent terminus-2 \
  --model openrouter/nex-agi/nex-n2.5-pro:free \
  --ak reasoning_effort=max \
  --max-retries 5 --n-concurrent 1 --n-attempts 1 \
  --timeout-multiplier 1.0 \
  --jobs-dir experiments/harbor_nex/jobs \
  --job-name smoke-nex-twosumbest-valid --yes \
  > experiments/harbor_nex/smoke-nex-twosumbest-valid.log 2>&1 &
```

Full job (`experiments/harbor_nex/nex-full.config.json`): 9 tasks × `--n-attempts 2` = 18 trials, `--n-concurrent 2`, `--max-retries 5`, agent timeout 10800 s (`task.toml`). Job dir `experiments/harbor_nex/jobs/nex-full/`. Wall **97.5 min** (00:16:39–01:54:11 local).

Prior smoke `jobs/smoke-nex-twosumbest` died SIGTERM during image build (foreground timeout). `smoke-nex-twosumbest-rerun` passed in 4.1 min **without a bug** — the previous task snapshot had clean `seen[k-v]` (git-archive mtimes, patch not applied). Task was rebuilt; numbers below are the **valid** smoke.

`discover_packages_for_tests` now skips nested `go.mod` trees (`integration_tests/`). Without that, GetGlobalConfig’s instruction.md was a module-prefix error instead of f2p output.

## Verifier table

Alt/cheat applied on the **buggy** tree. Alt must pass f2p; cheat must fail f2p. All six valid client-go bugs survive (NewBackofferWithVars from the prior pilot). No exclusions.

| bug | alternative | f2p + alt | cheat | f2p + cheat |
|---|---|---|---|---|
| NewBackofferWithVars | `b := NewBackoffer(...); return b.withVars(vars)` | **accept** | `if maxSleep==4 { withVars }` | **reject** |
| NewRequest | assign `req.Context = ctxs[0]` after constructing `*Request` | **accept** (`internal/locate` 14 s) | restore context only for `CmdEmpty` | **reject** |
| NewRegionRequestSender | construct sender then `s.client = client` | **accept** | set client only if `regionCache==nil` | **reject** |
| GetGlobalConfig | type-assert `globalConf.Load().(*Config)` with `DefaultConfig` fallback | **accept** | return stored cfg only if `Path=="__never_used_path__"` | **reject** (`TestRawKV` 49 s) |
| IsErrNotFound | `err==nil → false` then `errors.Is` | **accept** | special-case sentinel string, else still invert | **reject** |
| IsFakeRegionError | nil-check + `len(CurrentRegions)==0` | **accept** | treat `GetNotLeader()!=nil` as fake, else still invert | **reject** |

Patches: `experiments/codegraph_bugs/bugs/client-go/<Symbol>.{alt,cheat}.patch`. Logs: `experiments/harbor_nex/verifier/`.

Daily controls (prior): TwoSumBest / Match already had alt/cheat; TwoSumBrute used as an easy Harbor control without a new alt/cheat this session.

## Docker fail-to-pass (before launch)

Each task image `FROM golang:1.23`, modules downloaded at build, tests = exact f2p names. Buggy tree must fail; restoring the original mutated file must pass.

| task | buggy | fixed |
|---|---|---|
| dailycodingproblem-go-twosumbest | fail `TestTwoSumsSmall/Best`, `TestTwoSumsLarge/Best` | pass |
| dailycodingproblem-go-twosumbrute | fail `TestTwoSumsSmall/Brute` | pass |
| dailycodingproblem-go-match | fail `TestMatch` | pass |
| client-go-iserrnotfound | fail 4 f2p | pass |
| client-go-newrequest | fail `TestRegionRequestToThreeStores` | pass |
| client-go-newregionrequestsender | fail `TestRegionRequestToThreeStores` | pass |
| client-go-isfakeregionerror | fail 2 f2p | pass (1st restore hit a flake `TestTiKVRecoveredFromDown` nil panic; retry pass) |
| client-go-newbackofferwithvars | fail 2 f2p | pass |
| client-go-getglobalconfig | fail 4 f2p | pass |

## Smoke (valid)

| item | value |
|---|---|
| job | `smoke-nex-twosumbest-valid` |
| result | **pass** reward 1.0 |
| wall | 104 s (env 3 s, agent setup 6 s, agent 75 s, verifier 8 s) |
| tokens | in 36411 / cache 26880 / out 1639 |
| API calls | 6; times ms: 4916, 3262, 8278, 39821, 4274, 10365 |
| retries / 429 / 502 | 0 / 0 / 0 |
| note | litellm “model isn’t mapped” fallback context 1e6 on every call (not an API error) |

## Full job — per-task outcomes

Attempt 2 of every task, plus attempt 1 of NewRegionRequestSender / NewBackofferWithVars / GetGlobalConfig, died on OpenRouter **429 daily quota** (`X-RateLimit-Limit: 1000`, remaining 0, reset 2026-09-19 00:00 UTC). Harbor `n_retries=60`. Harbor retries **overwrite** the trial dir: NewRegionRequestSender attempt 1 was mid-debug for ~40 min, then the stored result is a 44 s RateLimitError with 0 tokens.

| task | att.1 | att.2 | wall s | API calls | tokens in/cache/out | notes |
|---|---|---|---:|---:|---|---|
| dailycodingproblem-go-twosumbest | **pass** | 429 | 71 | 6 | 30870 / 20352 / 1235 | same fix as smoke |
| dailycodingproblem-go-twosumbrute | **pass** | 429 | 137 | 9 | 71656 / 58944 / 3860 | |
| dailycodingproblem-go-match | **pass** | 429 | 318 | 11 | 84772 / 69824 / 13910 | |
| client-go-iserrnotfound | **fail** | 429 | 271 | 8 | 67985 / 51712 / 3865 | verifier still `not exist` on f2p |
| client-go-isfakeregionerror | **pass** | 429 | 1386 | 59 | 2.09M / 2.03M / 35k | hops=1, 16 impact files |
| client-go-newrequest | **pass** | 429 | 2541 | 76 | 3.37M / 3.28M / 60k | hops=1, 43 impact files |
| client-go-newregionrequestsender | 429† | 429 | 44* | 0* | 0* | hops=1, 33 files; †long live attempt overwritten |
| client-go-newbackofferwithvars | 429 | 429 | 46* | 0* | 0* | hops=1, 69 files; never a scored episode |
| client-go-getglobalconfig | 429 | 429 | 45* | 0* | 0* | hops=2, 66 files; never a scored episode |

\* retry-replacement record (env+first LLM 429), not the live attempt.

Harbor eval: `n_trials=6` scored, `n_errors=12`, mean reward **0.278** (5/18), **pass@2 = 0.556** (5/9 tasks had a passing attempt).

## Stability

| metric | value |
|---|---|
| HTTP 429 in `job.log` | **288** (`free-models-per-day-high-balance`) |
| HTTP 502 | **0** |
| Harbor trial retries | **60** |
| RateLimitError trials | **12** |
| scored exceptions | none (the 1 fail is a real miss, reward 0) |
| litellm unmapped-model warning | every episode; fallback ctx 1e6 |
| per-call latency (169 successful calls) | p50 **13.6 s**, p95 **53.5 s**, mean 19.3 s, min 2.4 s, max 129 s |
| job tokens | in 5.72M / cache 5.51M / out 118k / `cost_usd` null |

Latency is dominated by IsFake (59 calls) and NewRequest (76). Daily p50 is ~5–10 s (see smoke).

## Nex vs codegraph difficulty prior

Priors from `experiments/codegraph_bugs/RESULT.md` (valid cross-file only). Nex column is **attempt 1 scored outcome**.

| symbol | hops | impact files | prior | Nex att.1 |
|---|---:|---:|---|---|
| IsFakeRegionError | 1 | 16 | cheap end | **pass**, 23 min, 59 calls |
| IsErrNotFound | 1 | 19 | cheap end | **fail**, 4.5 min, 8 calls |
| NewRegionRequestSender | 1 | 33 | mid | 429 (unscored; ~40 min live work lost) |
| NewRequest | 1 | 43 | mid | **pass**, 42 min, 76 calls |
| GetGlobalConfig | 2 | 66 | fat blast | 429 before first token |
| NewBackofferWithVars | 1 | 69 | fat blast | 429 before first token |

Daily relaxed controls (not in the cross-file yield): all **pass** in 1–5 min.

No monotone “more hops / more impact files → harder for Nex” on this sample: the two scored 1-hop client-go bugs split (IsFake pass, IsErrNotFound fail), and the fat-blast symbols never got a scored attempt because the quota died first. The only clean signal is **easy intra-package daily vs everything else in wall time**.

## Wall-time cost per rollout

| class | typical wall | API calls | usable as a cheap probe? |
|---|---|---|---|
| daily (TwoSum/Match) | 1–5 min | 6–11 | yes |
| client-go 1-hop (scored) | 23–42 min | 59–76 | only with quota headroom |
| client-go after free-tier cap | 44 s then RateLimitError | 0 | no |

At 2 concurrent and reasoning=max, a 9-task × 2-attempt job **burns the 1000/day free cap** before attempt 2. Nex-free is a fine **spot check** on a single easy task (smoke 104 s, $0) and can solve some injected client-go bugs, but it is **not stable enough** to map codegraph priors onto pass rates without a paid OpenRouter key (or waiting for 00:00 UTC reset).

## Reproduce

```bash
uv run pytest tests/test_harbor_tasks.py   # 10 passed
uv run ruff check src/openswe_traces/synth/harbor_tasks.py tests/test_harbor_tasks.py
# tasks already in experiments/harbor_nex/tasks/; docker verify:
#   bash experiments/harbor_nex/verifier/docker_verify.sh <task> <image> <host_repo> <orig_file...>
```
