# End-to-end trial model for Open-SWE-Traces

One **trial** = one agent attempt to fix one GitHub issue.

## Identity

| Field | Role |
|---|---|
| `trajectory_id` | Primary key for the trial run |
| `instance_id` | Task / issue id (same task can have many trials) |
| `repo`, `language`, `category` | Task context |

## Trial lifecycle

```
TASK SETUP
  system prompt + tool definitions (tools[])
  user message = issue/PR description

AGENT LOOP (messages[2..n-1])
  assistant → tool_call(s) → tool observation → … repeat

SUBMIT
  final bash/edit producing model_patch

EVALUATE
  resolved ∈ {1 success, 0 fail, -1 unknown}
  compare model_patch vs reference_patch (gold)
```

## Turn types (one row in `trace_turns`)

| role | Meaning |
|---|---|
| `system` | Agent instructions + environment rules |
| `user` | Issue statement (the task) |
| `assistant` | LLM reply; may include `tool_calls` |
| `tool` | Environment feedback (bash output, file contents, errors) |

Typical loop: `assistant → tool → assistant → tool → …`

## Concrete example (shortest resolved Python mini-swe-agent trial)

`instance_id`: `astropenguin__xarray-dataclasses-8` · 19 turns · 1 tool (`bash`)

1. **system** — agent rules
2. **user** — PR description / bug statement
3. **assistant** — `bash: ls -la` (explore repo)
4. **tool** — directory listing
5. **assistant** — `cat pyproject.toml`, `cat __init__.py`, `cat tests/...` (read context)
6. **tool** — file contents
7. **assistant** — `sed` edit to bump version
8. **tool** — success
9. **assistant** — verify + `git diff > patch.txt`
10. **assistant** — `echo COMPLETE_TASK_AND_SUBMIT_FINAL_OUTPUT && cat patch.txt` (submit)

**Outcome**: `resolved=1`, model patch edits `xarray_dataclasses/__init__.py`.

## What to store for analysis / early prediction

Per **trial** (`trial_summary`):

- `num_messages`, `outcome_label`, patch sizes, harness, teacher

Per **turn** (`trace_turns` — sample or filter heavily):

- `turn_idx`, `role`, `has_tool_call`, `content_len`, `tool_name`

Features for turn *k* prediction (computed downstream):

- edits so far, tests run, unique files touched, repeated commands, tool errors

## DuckDB views

- `traces` — one row per trial (raw messages kept)
- `trial_summary` — trial facts without unpacking messages
- `trace_turns` — one row per message (expensive at full scale; use LIMIT/WHERE)
