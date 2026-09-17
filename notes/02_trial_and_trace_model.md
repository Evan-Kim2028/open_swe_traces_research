# Trial and trace model

See also: `analytics/research/trace_trial_model.md`

## One trial = one agent attempt on one issue

```
TRIAL
├── task_id      → instance_id
├── trial_id     → trajectory_id (UUID)
├── context      → repo, language, category, harness, teacher_model
│
├── SETUP
│   ├── messages[0]  system prompt
│   ├── messages[1]  user = issue / PR description
│   └── tools[]      available functions (bash, file edit, …)
│
├── LOOP
│   ├── assistant  → reasoning + tool_calls
│   └── tool       → bash output / file contents / errors
│
├── SUBMIT
│   └── model_patch (metadata.model_patch)
│
└── OUTCOME
    ├── reference_patch (gold)
    └── resolved ∈ {1, 0, -1}
```

## Turn types

| role | Meaning |
|---|---|
| `system` | Agent instructions |
| `user` | Issue statement |
| `assistant` | LLM reply; may include `tool_calls` |
| `tool` | Environment feedback |

Typical loop: `assistant → tool → assistant → tool → …`

## Example (shortest resolved Python mini-swe-agent trial)

`instance_id`: `astropenguin__xarray-dataclasses-8` · ~19 turns · tool: `bash` only

1. `ls` → explore repo
2. `cat` files → read context
3. `sed` → apply fix
4. `git diff` → produce patch
5. `COMPLETE_TASK_AND_SUBMIT_FINAL_OUTPUT` → submit

## DuckDB views

| View | Use |
|---|---|
| `traces` | One row per trial (includes `messages`) |
| `trial_summary` | Trial facts without unpacking messages (fast) |
| `trace_turns` | One row per message (**expensive** at full scale) |
| `turn_sample` | Materialized sample in `outputs/turn_sample.parquet` |

## Turn-level features (for early prediction)

Extracted by `scripts/extract_turn_sample.py`:

- `turn_idx`, `pct_through`, `role`, `content_len`
- `has_tool_call`, `is_edit_command`, `is_test_command`
- Cumulative: `cum_edits`, `cum_tests`, `cum_tool_errors`, `cum_tool_calls`

Early sample hint (15 trials): successes had fewer edits by 75% through run; failures kept editing and erroring.
