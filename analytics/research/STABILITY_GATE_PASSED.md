# Stability gate: PASS

Measured over a rolling 6h window by `scripts/ops/stability_gate.py`.

```
over-cap trials      2/47 = 4%     (need <5%)
devin trials sampled 20 of 20      (sample size, not concurrency)
new failure modes    0             (need 0)
```

## What the gate means

Scaling is now a budget question rather than an engineering one. The three
criteria were chosen because each one, when it failed, was hiding a defect that
more spend would simply have multiplied.

**over-cap trials** — a rung answers one question and needs one verdict. Three is
the cap; beyond that the trial buys nothing. This sat at 25% for most of the run.
Two causes: contract repairs resetting staleness so decided units re-entered the
queue, and two launchers each counting Devin occupancy their own way. 272 over-cap
trials were run before the fix, 0 after.

**devin trials sampled** — 20 verdicts is the smallest sample in which a
systematic Devin failure would show. Reaching it took until now not because Devin
is slow but because `reap_wedged.sh` was killing live trials: it judged a
container by local CPU and log growth, and Devin runs its model on Devin's
servers, so a thinking container is silent on both. Nine kills, four of them L2s.

**new failure modes** — every distinct error signature seen in the window must
already be in `outputs/supervisor/known_failures.json`. A new one means the
pipeline is still teaching us something, and spending into that is spending into
a moving target.

## What the gate does not mean

- It is a stability claim about the *pipeline*, not a quality claim about the
  *bank*. Certificate quality is the verifier's job (A1/A3/A12, B2, B7).
- The trial-duration statistics gathered before the reaper fix are censored:
  anything quiet past 45 minutes was killed, so the observed 21-min median and
  53-min max are lower bounds. Re-measure before quoting an ETA.
- Cross-repo remains a known limitation (`KNOWN_LIMITATION_cross_repo.md`).

## State at the moment it passed

```
units                 345        certified          148
trial dirs           1246          single-solver    133
  with a verdict     1156          cross-solver      15
too-easy              136          flipped above L2   9
non-flipping            0        escalatable         21
trials/certified      7.8        tokens 2.32B  cost $519.49
```
