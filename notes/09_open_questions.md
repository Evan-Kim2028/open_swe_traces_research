# Open questions

Move answers to `analytics/research/LOG.md` when resolved.

1. **Early prediction:** At turn *k*, what signals best predict `resolved=1`?  
   Related: [2607.17205](https://arxiv.org/abs/2607.17205), SWE-Prime process quality.

2. **Quality ranking:** Can a heuristic score (efficiency + patch overlap + no test edits) rank traces better than `resolved` alone?  
   Related: [SWE-Prime](https://arxiv.org/abs/2608.27449).

3. **Failure modes:** What share of failures are empty patch vs wrong file vs test-suite edits vs tool loops?

4. **Teacher effect:** Do MiniMax thinking traces differ in turn count / resolve rate from Qwen non-thinking?  
   Open-SWE-Traces paper claims thinking traces are more turn-efficient ([2606.16038 Table 1](https://arxiv.org/abs/2606.16038)).

5. **Training subset:** Resolved-only vs all vs quality-top-k — which wins on a small SFT run?  
   Open-SWE-Traces: full > resolved-only. SWE-Prime: top-10% > all resolved. SWE-smith: resolved-only.

6. **Unknown labels:** How to treat `resolved=-1` (especially OpenHands)? Exclude from training labels or impute?

7. **Cross-harness:** Do traces from one harness transfer to eval on another?  
   Open-SWE-Traces ablation: yes, with penalty ([§4.1](https://arxiv.org/abs/2606.16038)).
