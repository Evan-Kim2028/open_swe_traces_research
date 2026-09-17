# Dataset and trial model

## Open-SWE-Traces (primary source)

- **Paper:** [arXiv:2606.16038](https://arxiv.org/abs/2606.16038)
- **Dataset:** [nvidia/Open-SWE-Traces](https://huggingface.co/datasets/nvidia/Open-SWE-Traces)
- **Scale:** ~511k trajectories, ~43 GB, 9 languages, OpenHands + SWE-agent (+ v1.1/v1.2 additions)
- **License:** CC BY 4.0; underlying repos MIT/Apache/BSD

### Key fields

| Field | Meaning |
|---|---|
| `trajectory_id` | Unique trial run |
| `instance_id` | Task/issue (many trials per task) |
| `messages` | Full chat: system, user, assistant, tool |
| `tools` | Tool definitions (JSON strings) |
| `resolved` | 1 = fixed, 0 = failed, -1 = unknown |
| `metadata.reference_patch` | Gold diff |
| `metadata.model_patch` | Agent's diff |

### Trial lifecycle

```
TASK SETUP → system + tools + user issue
AGENT LOOP → assistant → tool → … repeat
SUBMIT → model_patch
EVALUATE → resolved + compare to gold
```

See `analytics/research/trace_trial_model.md` for full model.

## EDA findings (partial → full download)

- ~348k+ trials at last check; download completed per user
- mini-swe-agent ~41% resolved on Python; OpenHands many `resolved=-1`
- Successes tend shorter (avg ~96 turns) vs failures (~123+)
- bug-fix resolves more often than feature-request

## External dataset candidates

See `analytics/research/external_datasets.md` and `notes/05_external_datasets.md`.

## Sources

- [Open-SWE-Traces HF card](https://huggingface.co/datasets/nvidia/Open-SWE-Traces)
- [SWE-rebench-V2](https://huggingface.co/datasets/nebius/SWE-rebench-V2) (upstream tasks)
- Local: `evaltrials/registry/scores-and-traces.yaml` (row counts verified 2026-09-14)
