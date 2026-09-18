# Synthetic tasks: placing new tasks on the IRT difficulty scale with ~2 cheap rollouts

2026-09-18. Depends on `outputs/task_irt.parquet` (36k anchored tasks, b_2pl) and
`outputs/combo_ability.parquet` (theta per harness/teacher). See `irt_summary.md`,
`cheap_difficulty_summary.md`.

## Why this matters

Task-only features predict 10% of difficulty variance. Difficulty must be measured by rollouts.
Full measurement is ~12 teacher rollouts per task. This workflow gets a usable estimate from ~2
rollouts of ONE cheap model, because the cheap model's ability theta is already known from the bank.

## Step 3 — the posterior (the math)

For a new task with unknown difficulty b, prober with known ability theta_p and (assume) a = 1:

    prior      p(b)        = empirical distribution of b_2pl over the bank (or N(mean, sd) fit to it)
    likelihood p(y | b)    = sigmoid(theta_p - b)^y * (1 - sigmoid(theta_p - b))^(1-y),   y in {0,1}
    posterior  p(b | y)   ∝  p(b) * p(y | b)

Compute on a grid of b (e.g. 400 points from -3 to 3). Output per task:

- estimate  = posterior mean of b
- width     = posterior sd of b
- P(bucket) = posterior mass inside each bucket's b-range (cuts from irt_summary: -0.874, -0.178, +0.531, +1.234)

With several outcomes (possibly from different probers) multiply the likelihoods. This is the whole
of step 3; it is a dozen lines of numpy and it is exact for 1PL.

## Step 4 — the decision rule (when to spend a second rollout)

Step 3 gives numbers; step 4 decides whether to pay for more.

- If max_bucket P(bucket) >= 0.8: stop, assign that bucket.
- Else: one more rollout. Choose the prober whose theta is closest to the current posterior mean
  (Fisher information is maximal when theta = b). With a single prober, just roll it again.
- Cap at 4 rollouts; assign the max-probability bucket at the cap.

Expected cost with one prober (from the adaptive-sampling simulation on the bank): ~2.3 rollouts
per task with the 2-agree rule, ~5.4 with an 80% interval rule. The posterior rule sits between:
all_pass/all_fail tasks settle in 1–2, mid tasks take 3–4.

Breakdown: step 3 = compute belief; step 4 = act on belief. Step 3 never costs a rollout; step 4 is
the only place rollouts are spent, and only on tasks whose belief is still ambiguous.

## Step 5 — misfit flags (quality control)

- Prober passes but gold-patch tests are marginal, or the trace touches `.git` history: suspect leak.
- Outcome pattern contradicts ability order across probers (weak passes, strong fails): suspect grader.
- Posterior width does not shrink after 4 rollouts: task is a coin flip for everyone (low a); low value.

## Prober choice

- `minisweagent/qwen36_27b` is already calibrated (theta = -0.27, 2PL). Zero calibration cost.
- Any other cheap/free model: run it once on ~200 bank tasks spanning b, fit its theta by holding
  all b fixed (one scalar MLE). Then use it as above.

## Where the synthetic tasks come from

SWE-smith (Yang et al. 2025): procedural + LM bug injection into repos that already have a working
Docker environment and test suite; each task ships fail-to-pass tests, so grading is automatic.
The SWE-rebench-V2 instances behind Open-SWE-Traces come with environment specs, so the 3k repos in
this corpus are candidate hosts for new bugs. Sanity filter before any rollout: gold patch passes,
empty patch fails.

## Not yet built

- `src/openswe_traces/posterior.py` (steps 3–4 on the grid) — trivial given the bank.
- A rollout runner for the prober (OpenRouter or Bonsai-on-Kaggle) — external cost.
- SWE-smith-style generation against SWE-rebench-V2 environments — moderate work.
