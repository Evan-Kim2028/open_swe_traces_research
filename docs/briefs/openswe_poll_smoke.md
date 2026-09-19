Repo: /home/evan/Documents/open_swe_traces_research (work only under experiments/kaggle_smoke).
A Kaggle script kernel `evandekim/openswe-smoke-qlora` was just pushed. Kaggle CLI is invoked as:
  uv run --project /home/evan/Documents/kaggle_e4b kaggle <args>
Task:
1. Poll `kaggle kernels status evandekim/openswe-smoke-qlora` every 60s (use sleep 60 in a loop; up to 100 min) until status contains complete/error/cancel.
2. Then `kaggle kernels output evandekim/openswe-smoke-qlora -p experiments/kaggle_smoke/out`.
3. If smoke_metrics.json exists, write experiments/kaggle_smoke/RESULT.md: GPU, tok_per_s, sup_tok_per_s, sec_per_trace, est_hours_per_10k_traces, peak_mem_gb, avg_len, n_truncated/n_examples, first vs last loss, and a budget table: hours needed for 5k/10k/20k/40k traces at 1 epoch given sec_per_trace, and whether each fits a 12h Kaggle session.
4. If the kernel errored, pull the log (`kaggle kernels output` gives the .log), put the last 60 lines and your diagnosis of the root cause in RESULT.md. Do NOT re-push the kernel (GPU quota is scarce).
Never use pkill/pgrep patterns that contain this script's own name.
