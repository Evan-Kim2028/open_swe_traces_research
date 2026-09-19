Repo: /home/evan/Documents/open_swe_traces_research. Data: traces_data/data/<harness>/<teacher>/<source>/*.parquet (212 files, 42GB, 511,668 rows). Schema: instance_id, repo, language, trajectory_id, messages (list of struct role/content/reasoning_content/think/tool_calls), tools, resolved (1/0/-1), metadata (struct with category, reference_patch{patch,num_modified_files,num_modified_lines}, model_patch{...}), hf_dataset_name. Use `uv run python` (duckdb, polars, pyarrow installed). Existing DuckDB helpers: scripts/duckdb_init.py, scripts/duckdb_query.py, analytics/schema/*.sql. Read those first and follow the conventions.

Goal: zero-GPU per-trajectory structural features for ranking traces before SFT. Write scripts/proxy_features.py that produces outputs/proxy_features.parquet with one row per trajectory_id and columns:
  harness, teacher, source (parsed from file path), instance_id, repo, language, category, resolved,
  n_messages, n_assistant_turns, n_tool_calls, assistant_chars, reasoning_chars, tool_obs_chars,
  n_distinct_tool_commands, repeat_call_rate (1 - distinct/total tool call argument strings),
  max_consecutive_identical_calls,
  n_edit_calls (bash commands containing sed -i, cat >, tee, str_replace_editor, edit, apply_patch, or an edit-type tool),
  n_test_calls (pytest, go test, cargo test, npm test, jest, phpunit, mvn test),
  model_patch_files, model_patch_lines, gold_patch_files, gold_patch_lines,
  patch_file_jaccard (Jaccard of file paths touched in model_patch vs reference_patch, parsed from `diff --git a/X b/X` or `+++ b/X` lines),
  ends_with_submit (last assistant turn contains submit / finish / done marker).
Constraints: stream file-by-file (do not load all 42GB into memory; MemoryMax ~24GB on this box), write partitioned or append per file, resume-safe (skip files already done), log progress. Run it on ONE file first, print head and a sanity summary, then run the full corpus in the background with nohup and log to outputs/proxy_features.log. When finished, write analytics/research/proxy_features_summary.md with: per harness/teacher means of the key features, correlation of each feature with `resolved` (only rows where resolved in (0,1)), and the 5 features most predictive of resolved by simple logistic regression (sklearn, standardized). Commit nothing.
Never use pkill/pgrep patterns that contain this script's own name.
