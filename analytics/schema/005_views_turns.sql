-- Turn-level view for one trial = one trajectory.
-- Each row is one message (turn) within a trial run.
-- Heavy on full table scans — use with WHERE / LIMIT during exploration.

CREATE OR REPLACE VIEW trace_turns AS
SELECT
    t.trajectory_id,
    t.instance_id,
    t.repo,
    t.language,
    t.harness,
    t.teacher_model,
    t.resolved,
    t.category,
    turn_idx,
    msg.role AS role,
    coalesce(length(msg.content), 0) AS content_len,
    json_extract_string(to_json(msg), '$.name') AS tool_name,
    (
        msg.role = 'assistant'
        AND json_extract(to_json(msg), '$.tool_calls') IS NOT NULL
    ) AS has_tool_call,
    msg
FROM traces AS t,
     unnest(t.messages) WITH ORDINALITY AS u(msg, turn_idx);

-- Trial-level summary (cheap; no message unpack beyond len()).

CREATE OR REPLACE VIEW trial_summary AS
SELECT
    trajectory_id,
    instance_id,
    repo,
    language,
    harness,
    teacher_model,
    source_dataset,
    resolved,
    category,
    num_messages,
    gold_files,
    gold_lines,
    model_files,
    model_lines,
    coalesce(model_patch, '') = '' AS empty_model_patch,
    CASE
        WHEN resolved = 1 THEN 'success'
        WHEN resolved = 0 THEN 'failed'
        ELSE 'unknown'
    END AS outcome_label
FROM traces;
