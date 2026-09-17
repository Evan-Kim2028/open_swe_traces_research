# Open questions

Track hypotheses here; move answers to `LOG.md` when resolved.

1. **Early prediction:** At turn *k*, what signals best predict `resolved=1`?
2. **Quality ranking:** Can a heuristic score (efficiency + patch overlap + no test edits) rank traces usefully?
3. **Failure modes:** What % of failures are empty patch vs wrong file vs test-suite edits vs tool loops?
4. **Teacher effect:** Do MiniMax thinking traces differ in turn count / resolve rate from Qwen non-thinking?
5. **Training subset:** Which filtered slice (resolved-only vs all, Python-only vs multilingual) is best for SFT?
