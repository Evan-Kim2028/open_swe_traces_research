# End-to-end trial model

One **trial** = one agent attempt to fix one GitHub issue.  
Source schema: [Open-SWE-Traces dataset card](https://huggingface.co/datasets/nvidia/Open-SWE-Traces).

## Identity

| Field | Role |
|---|---|
| `trajectory_id` | Primary key for the trial run (UUID) |
| `instance_id` | Task / issue id (many trials per task) |
| `repo`, `language`, `license`, `category` | Task context |
| `harness`, `teacher_model` | Parsed from parquet path (`data/{harness}/{teacher}/…`) |

## Trial lifecycle

```
TASK SETUP
  messages[0]  system prompt
  messages[1]  user = issue / PR description
  tools[]      bash, file edit, etc.

AGENT LOOP
  assistant  →  tool_calls
  tool       →  environment observation
  (repeat)

SUBMIT
  metadata.model_patch  agent's final diff

EVALUATE
  resolved ∈ {1, 0, -1}
  metadata.reference_patch  gold diff
```

## Turn types (`trace_turns` / `turn_sample`)

| role | Meaning |
|---|---|
| `system` | Agent instructions |
| `user` | Issue statement |
| `assistant` | LLM reply; may include `tool_calls` |
| `tool` | Bash output, file read, errors |

Typical loop: `assistant → tool → assistant → tool → …`

## Example: short resolved Python trial (mini-swe-agent)

`instance_id`: `astropenguin__xarray-dataclasses-8` · ~19 turns · tool: `bash`

1. `ls -la` — explore repo  
2. `cat` project files — read context  
3. `sed` — apply fix  
4. `git diff` — produce patch  
5. `COMPLETE_TASK_AND_SUBMIT_FINAL_OUTPUT` — submit  

Outcome: `resolved=1`; model patch edits `xarray_dataclasses/__init__.py`.

## DuckDB views (repo)

| View | Use |
|---|---|
| `traces` | One row per trial; full `messages` |
| `trial_summary` | Trial facts without message unpack (fast) |
| `trace_turns` | One row per message (expensive at full scale) |
| `turn_sample` | Materialized sample parquet for modeling |

## Features for analysis / early prediction

**Trial-level:** `num_messages`, `outcome_label`, patch sizes, harness, teacher.

**Turn-level:** `turn_idx`, `pct_through`, `cum_edits`, `cum_tests`, `cum_tool_errors`, `has_tool_call`.

Script: `scripts/extract_turn_sample.py` (DuckDB streaming, 4GB cap).
