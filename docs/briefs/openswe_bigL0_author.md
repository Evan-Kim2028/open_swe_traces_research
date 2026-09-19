Repo: /home/evan/Documents/open_swe_traces_research. Read AGENTS.md, analytics/research/verifier_rules.md (ladder L0-L6, rules incl. B4, A12, C5, C6), experiments/harbor_nex/ITER_9.md and ITER_8.md (the two largest excisions: keyspacecodec 906 lines / 5 files, batchcmds 221 lines / 4 files), experiments/harbor_nex/tasks_obf/client-go-keyspacecodec-obf and client-go-batchcmds-obf (obfuscated trees + gold patches), src/openswe_traces/synth/{affordance,rules}.py, experiments/harbor_nex/base/Dockerfile (base image ladder-base:client-go-obf).
You are the TASK AUTHOR only. Do NOT write any verifier test. For each of the two units (keyspacecodec-obf, batchcmds-obf): (1) produce the excised tree (apply the excision as in the obf task; interface removed as in ITER_8/9); (2) write api.md: the exported API surface of the unit as a caller sees it (names, signatures, doc comments that exist in the repo) and the list of pre-existing callers; (3) write contract.md: the full behavioral contract (L2 wording) with a coverage table against the ORIGINAL in-tree tests (test -> sentence), no function names; (4) write bugreport.md: the L0 instruction — a user-style report of what breaks with the unit missing (observable operation, wrong output/panic) + a reproduction command that will run the hidden suite via tests/test.sh (the verifier author will fill tests/); (5) gold.patch (restores the unit, must NOT touch any *_test.go), cheat.patch (hardcodes outputs for the contract examples); (6) difficulty.md with a REQUIRED line exactly `predicted_flip: L<k>` (k in 0..6 — the ladder level you expect the solver's flip point to fall at) plus one short justification paragraph. L2 = full contract; L0 = bug report only. Put everything under experiments/harbor_nex/tasks_bigL0/<unit>/_author/ and write AUTHOR.md. Do not build Docker images, do not write tests. Print AUTHOR.md as your last message.

## Required in difficulty.md

Every authored unit's `_author/difficulty.md` MUST contain a line exactly of the form:

predicted_flip: L<k>

where k is an integer 0..6. Surrounding prose is allowed. Example:

```
# Difficulty
predicted_flip: L2
Justification: single-file missing-return; in-tree tests already describe it.
```

