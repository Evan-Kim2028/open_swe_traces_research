# End-to-end trial model

One **trial** = one agent attempt to fix one GitHub issue.

DuckDB views: `trial_summary` (cheap aggregates), `trace_turns` (expensive, sample only), `turn_sample` (materialized sample parquet).

## Identity

| Field | Role |
|---|---|
| `trajectory_id` | Primary key for the trial run |
| `instance_id` | Task / issue (many trials per task possible) |
| `repo`, `language`, `category` | Task context |
| `harness`, `teacher_model` | Parsed from parquet file path |

## Lifecycle

```
TASK SETUP
  system prompt + tools[]
  user message = issue / PR description

AGENT LOOP
  assistant → tool_call(s) → tool observation → repeat

SUBMIT
  model_patch (agent's final diff)

EVALUATE
  resolved ∈ {1, 0, -1}
  compare model_patch vs reference_patch (gold)
```

## Turn types (`trace_turns` / `turn_sample`)

| role | Meaning |
|---|---|
| `system` | Agent instructions |
| `user` | Issue statement |
| `assistant` | LLM reply; may include `tool_calls` |
| `tool` | Environment feedback (bash output, errors) |

Typical loop: `assistant → tool → assistant → tool → …`

## Concrete example (shortest resolved Python mini-swe-agent trial)

- `instance_id`: `astropenguin__xarray-dataclasses-8`
- 19 turns, single tool (`bash`)
- Flow: `ls` → read files → `sed` edit → verify → `git diff` → submit
- `resolved=1`

## Turn-level features (for early prediction)

Per turn in `turn_sample`:
- `turn_idx`, `pct_through`, `role`, `content_len`
- `has_tool_call`, `is_edit_command`, `is_test_command`
- Cumulative: `cum_edits`, `cum_tests`, `cum_tool_errors`, `cum_tool_calls`

## Extraction script

```bash
uv run python scripts/extract_turn_sample.py --sample-size 30 --register-duckdb
```

Uses DuckDB streaming: locate files → per-file INSERT into temp table → COPY to parquet.

Source: `scripts/extract_turn_sample.py`, `analytics/schema/005_views_turns.sql`
