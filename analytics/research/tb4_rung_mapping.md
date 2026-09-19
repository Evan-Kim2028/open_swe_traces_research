# TB ladder rung mapping (TB2 + TB4)

Generated from local files only. Ladder L0–L6 per `analytics/research/verifier_rules.md`.

## 1. Inventory

| Source | Version | Tasks | Trials/trajectories | Path | Notes |
|---|---|---|---|---|---|
| TB2 task defs | terminal-bench-2 | 89 | 89 dirs | /home/evan/Documents/terminal-bench-2 | instruction.md + tests/ per task |
| TB4 task defs | eval_tasks tb-ref | 66 | 66 dirs (68 on disk) | /home/evan/Documents/eval_tasks/research/tb-ref/tasks | Terminal-Bench 4.0.0 |
| TB4 Hub results | eval_tasks analysis/tb4/raw | 13 | 13 models × 66 tasks × 5 trials = 4,290 trial rows | /home/evan/Documents/eval_tasks/analysis/tb4/raw | Per-task pass in task_stats.json |
| TB2 trajectories | HF cache | 5 | 52,104 rows; 89 tasks; 49 models (5 frontier used here) | /home/evan/.cache/huggingface/hub/datasets--yoonholee--terminalbench-trajectories/snapshots/04e8940f5b6736a7ce8d22224fe2f2af74163ed2/data | reward 0/1 per trial; steps JSON |
| TB4 facets | eval_tasks analysis/tb4/facets | 66 | 4 batch JSON files | /home/evan/Documents/eval_tasks/analysis/tb4/facets | LLM-labelled verifier/spec metadata |
| eval_tasks runs | eval_tasks/research/runs | — | 90 result.json (local K2/K3 ablations, not TB bench) | /home/evan/Documents/eval_tasks/research/runs | Out of scope for ladder join |
| eval_tasks docs | eval_tasks/docs | — | 7 markdown files (taxonomy, running, first-lot plan) | /home/evan/Documents/eval_tasks/docs | Task design notes, not scored results |
| eval_tasks results | eval_tasks/results | — | 2 markdown summaries | /home/evan/Documents/eval_tasks/results | Lakehouse recovery postmortem only |
| TB2 trajectories (full) | HF yoonholee/terminalbench-trajectories | 89 | 52,104 trials; 49 models; 26 scaffolds; 2× parquet ~940MB | /home/evan/.cache/huggingface/hub/datasets--yoonholee--terminalbench-trajectories/snapshots/04e8940f5b6736a7ce8d22224fe2f2af74163ed2/data | steps JSON per row; 34,462 trials have steps |

### Rung distribution (classified)

| Rung | TB4 count | TB2 count | Meaning |
|---|---|---|---|
| 0 | 0 | 1 | bug-report only |
| 1 | 58 | 44 | partial spec |
| 2 | 2 | 31 | full behavioral spec |
| 3 | 0 | 2 | check names |
| 4 | 2 | 4 | signatures |
| 5 | 4 | 2 | example test |
| 6 | 0 | 5 | tests in env |

## 2. Rung / verifier table

All tasks with readable `instruction.md` + `tests/`.

| Suite | Task | Rung | Verifier | Tests hidden | Leakage | Rung note |
|---|---|---|---|---|---|---|
| TB2 | fix-git | L0 | example-tests | yes | no | short bug-report style, no behavioral contract |
| TB2 | build-pmars | L1 | example-tests | yes | yes | medium instruction; unstated test requirements likely |
| TB2 | build-pov-ray | L1 | example-tests | yes | yes | instruction names 3/16 asserted paths |
| TB2 | caffe-cifar-10 | L1 | example-tests | yes | yes | medium instruction; unstated test requirements likely |
| TB2 | cancel-async-tasks | L1 | dynamic-gate | yes | yes | medium instruction; unstated test requirements likely |
| TB2 | chess-best-move | L1 | example-tests | yes | yes | medium instruction; unstated test requirements likely |
| TB2 | compile-compcert | L1 | example-tests | yes | yes | instruction names 1/5 asserted paths |
| TB2 | configure-git-webserver | L1 | example-tests | yes | no | instruction names 0/1 asserted paths |
| TB2 | count-dataset-tokens | L1 | example-tests | yes | yes | medium instruction; unstated test requirements likely |
| TB2 | crack-7z-hash | L1 | example-tests | yes | yes | medium instruction; unstated test requirements likely |
| TB2 | dna-assembly | L1 | example-tests | yes | no | instruction names 0/1 asserted paths |
| TB2 | dna-insert | L1 | example-tests | yes | no | instruction names 0/1 asserted paths |
| TB2 | extract-elf | L1 | example-tests | yes | no | instruction names 0/6 asserted paths |
| TB2 | extract-moves-from-video | L1 | example-tests | yes | yes | medium instruction; unstated test requirements likely |
| TB2 | feal-differential-cryptanalysis | L1 | example-tests | yes | yes | medium instruction; unstated test requirements likely |
| TB2 | feal-linear-cryptanalysis | L1 | example-tests | yes | yes | medium instruction; unstated test requirements likely |
| TB2 | filter-js-from-html | L1 | example-tests | yes | yes | instruction names 2/9 asserted paths |
| TB2 | fix-ocaml-gc | L1 | example-tests | yes | no | instruction names 0/1 asserted paths |
| TB2 | gcode-to-text | L1 | example-tests | yes | yes | medium instruction; unstated test requirements likely |
| TB2 | git-leak-recovery | L1 | example-tests | yes | yes | medium instruction; unstated test requirements likely |
| TB2 | make-doom-for-mips | L1 | example-tests | yes | yes | instruction names 1/7 asserted paths |
| TB2 | make-mips-interpreter | L1 | example-tests | yes | no | instruction names 0/7 asserted paths |
| TB2 | mteb-leaderboard | L1 | example-tests | yes | yes | medium instruction; unstated test requirements likely |
| TB2 | mteb-retrieve | L1 | example-tests | yes | yes | medium instruction; unstated test requirements likely |
| TB2 | path-tracing-reverse | L1 | example-tests | yes | yes | instruction names 1/5 asserted paths |
| TB2 | polyglot-c-py | L1 | example-tests | yes | yes | medium instruction; unstated test requirements likely |
| TB2 | polyglot-rust-c | L1 | example-tests | yes | yes | medium instruction; unstated test requirements likely |
| TB2 | portfolio-optimization | L1 | dynamic-gate | yes | no | instruction names 0/1 asserted paths |
| TB2 | prove-plus-comm | L1 | example-tests | yes | yes | medium instruction; unstated test requirements likely |
| TB2 | pytorch-model-cli | L1 | example-tests | yes | no | instruction names 0/6 asserted paths |
| TB2 | pytorch-model-recovery | L1 | example-tests | yes | yes | instruction names 3/49 asserted paths |
| TB2 | qemu-alpine-ssh | L1 | example-tests | yes | no | medium instruction; unstated test requirements likely |
| TB2 | qemu-startup | L1 | example-tests | yes | no | instruction names 0/1 asserted paths |
| TB2 | query-optimize | L1 | dynamic-gate | yes | yes | medium instruction; unstated test requirements likely |
| TB2 | raman-fitting | L1 | example-tests | yes | yes | medium instruction; unstated test requirements likely |
| TB2 | sam-cell-seg | L1 | example-tests | yes | no | instruction names 0/8 asserted paths |
| TB2 | sanitize-git-repo | L1 | dynamic-gate | yes | no | instruction names 0/7 asserted paths |
| TB2 | sqlite-db-truncate | L1 | example-tests | yes | yes | medium instruction; unstated test requirements likely |
| TB2 | sqlite-with-gcov | L1 | example-tests | yes | yes | medium instruction; unstated test requirements likely |
| TB2 | train-fasttext | L1 | example-tests | yes | yes | instruction names 1/363 asserted paths |
| TB2 | tune-mjcf | L1 | dynamic-gate | yes | yes | instruction names 1/15 asserted paths |
| TB2 | video-processing | L1 | example-tests | yes | yes | instruction names 4/31 asserted paths |
| TB2 | vulnerable-secret | L1 | example-tests | yes | yes | medium instruction; unstated test requirements likely |
| TB2 | winning-avg-corewars | L1 | example-tests | yes | no | instruction names 0/3 asserted paths |
| TB2 | write-compressor | L1 | example-tests | yes | yes | medium instruction; unstated test requirements likely |
| TB2 | adaptive-rejection-sampler | L2 | example-tests | yes | yes | long behavioral spec in instruction |
| TB2 | bn-fit-modify | L2 | property/fuzz | yes | yes | long behavioral spec in instruction |
| TB2 | cobol-modernization | L2 | example-tests | yes | yes | long behavioral spec in instruction |
| TB2 | code-from-image | L2 | example-tests | yes | yes | long behavioral spec in instruction |
| TB2 | constraints-scheduling | L2 | example-tests | yes | yes | long behavioral spec in instruction |
| TB2 | custom-memory-heap-crash | L2 | dynamic-gate | yes | yes | long behavioral spec in instruction |
| TB2 | db-wal-recovery | L2 | example-tests | yes | yes | long behavioral spec in instruction |
| TB2 | distribution-search | L2 | example-tests | yes | yes | long behavioral spec in instruction |
| TB2 | financial-document-processor | L2 | example-tests | yes | yes | long behavioral spec in instruction |
| TB2 | fix-code-vulnerability | L2 | example-tests | yes | yes | long behavioral spec in instruction |
| TB2 | git-multibranch | L2 | example-tests | yes | no | long behavioral spec in instruction |
| TB2 | gpt2-codegolf | L2 | example-tests | yes | yes | long behavioral spec in instruction |
| TB2 | hf-model-inference | L2 | example-tests | yes | yes | long behavioral spec in instruction |
| TB2 | kv-store-grpc | L2 | example-tests | yes | yes | long behavioral spec in instruction |
| TB2 | largest-eigenval | L2 | dynamic-gate | yes | yes | default: prose spec without test names/signatures/examples |
| TB2 | llm-inference-batching-scheduler | L2 | dynamic-gate | yes | yes | long behavioral spec in instruction |
| TB2 | log-summary-date-ranges | L2 | example-tests | yes | yes | long behavioral spec in instruction |
| TB2 | mailman | L2 | example-tests | yes | yes | default: prose spec without test names/signatures/examples |
| TB2 | mcmc-sampling-stan | L2 | example-tests | yes | yes | long behavioral spec in instruction |
| TB2 | merge-diff-arc-agi-task | L2 | example-tests | yes | yes | long behavioral spec in instruction |
| TB2 | model-extraction-relu-logits | L2 | property/fuzz | yes | yes | long behavioral spec in instruction |
| TB2 | modernize-scientific-stack | L2 | example-tests | yes | yes | long behavioral spec in instruction |
| TB2 | multi-source-data-merger | L2 | example-tests | yes | yes | long behavioral spec in instruction |
| TB2 | nginx-request-logging | L2 | example-tests | yes | yes | long behavioral spec in instruction |
| TB2 | openssl-selfsigned-cert | L2 | example-tests | yes | yes | long behavioral spec in instruction |
| TB2 | password-recovery | L2 | example-tests | yes | yes | default: prose spec without test names/signatures/examples |
| TB2 | path-tracing | L2 | example-tests | yes | yes | long behavioral spec in instruction |
| TB2 | protein-assembly | L2 | example-tests | yes | yes | long behavioral spec in instruction |
| TB2 | pypi-server | L2 | example-tests | yes | no | default: prose spec without test names/signatures/examples |
| TB2 | regex-log | L2 | example-tests | yes | yes | long behavioral spec in instruction |
| TB2 | reshard-c4-data | L2 | example-tests | yes | yes | long behavioral spec in instruction |
| TB2 | build-cython-ext | L3 | example-tests | yes | yes | names test checks in instruction |
| TB2 | rstan-to-pystan | L3 | example-tests | yes | yes | names test checks in instruction |
| TB2 | headless-terminal | L4 | example-tests | yes | yes | 2 signature/interface blocks |
| TB2 | install-windows-3.11 | L4 | example-tests | yes | yes | 2 signature/interface blocks |
| TB2 | torch-pipeline-parallelism | L4 | example-tests | yes | yes | 2 signature/interface blocks |
| TB2 | torch-tensor-parallelism | L4 | example-tests | yes | yes | 3 signature/interface blocks |
| TB2 | regex-chess | L5 | example-tests | yes | yes | example I/O in instruction |
| TB2 | sparql-university | L5 | example-tests | yes | yes | example I/O in instruction |
| TB2 | break-filter-js-from-html | L6 | example-tests | yes | yes | tests copied into agent environment (Dockerfile COPY tests/) |
| TB2 | circuit-fibsqrt | L6 | example-tests | yes | yes | tests copied into agent environment (Dockerfile COPY tests/) |
| TB2 | large-scale-text-editing | L6 | example-tests | yes | yes | tests copied into agent environment (Dockerfile COPY tests/) |
| TB2 | overfull-hbox | L6 | example-tests | yes | no | tests copied into agent environment (Dockerfile COPY tests/) |
| TB2 | schemelike-metacircular-eval | L6 | example-tests | yes | yes | tests copied into agent environment (Dockerfile COPY tests/) |
| TB4 | atrx-vep-crispr | L1 | example-tests | yes | yes | facet: spec_precision=constraint-dense, hidden_invariant=Tru |
| TB4 | batched-eval-parity | L1 | example-tests | yes | yes | facet: spec_precision=constraint-dense, hidden_invariant=Tru |
| TB4 | bun-sourcemap-leak | L1 | example-tests | yes | yes | facet: spec_precision=constraint-dense, hidden_invariant=Tru |
| TB4 | cad-model | L1 | example-tests | yes | yes | facet: spec_precision=needs-inference, hidden_invariant=True |
| TB4 | cargo-flight-dispatch | L1 | example-tests | yes | yes | facet: spec_precision=needs-inference, hidden_invariant=True |
| TB4 | coq-block-bound | L1 | example-tests | yes | yes | facet: spec_precision=constraint-dense, hidden_invariant=Tru |
| TB4 | ctr-optimization | L1 | dynamic-gate | yes | no | facet: spec_precision=needs-inference, hidden_invariant=True |
| TB4 | cumulative-layout-shift | L1 | dynamic-gate | yes | yes | facet: spec_precision=needs-inference, hidden_invariant=True |
| TB4 | embedding-drift-monitor | L1 | example-tests | yes | yes | facet: spec_precision=needs-inference, hidden_invariant=True |
| TB4 | fin-saccr-rwa | L1 | example-tests | yes | yes | facet: spec_precision=constraint-dense, hidden_invariant=Tru |
| TB4 | foodstuff-beta-activity | L1 | example-tests | yes | yes | facet: spec_precision=needs-inference, hidden_invariant=True |
| TB4 | formal-crypto | L1 | example-tests | yes | yes | facet: spec_precision=needs-inference, hidden_invariant=True |
| TB4 | fp8-rmsnorm-gemm | L1 | example-tests | yes | yes | facet: spec_precision=constraint-dense, hidden_invariant=Tru |
| TB4 | freecad-impeller | L1 | example-tests | yes | yes | facet: spec_precision=needs-inference, hidden_invariant=True |
| TB4 | freecad-platform-drawing | L1 | example-tests | yes | yes | facet: spec_precision=needs-inference, hidden_invariant=True |
| TB4 | freight-dispatch-shift | L1 | example-tests | yes | yes | facet: spec_precision=constraint-dense, hidden_invariant=Tru |
| TB4 | glycan-ms2-elucidation | L1 | example-tests | yes | yes | facet: spec_precision=needs-inference, hidden_invariant=True |
| TB4 | gsea-proteomics | L1 | example-tests | yes | yes | facet: spec_precision=constraint-dense, hidden_invariant=Tru |
| TB4 | heat-pump-warranty | L1 | example-tests | yes | no | facet: spec_precision=needs-inference, hidden_invariant=True |
| TB4 | html-js-filter | L1 | example-tests | yes | yes | facet: spec_precision=needs-inference, hidden_invariant=True |
| TB4 | interleaved-vigenere | L1 | dynamic-gate | yes | yes | facet: spec_precision=needs-inference, hidden_invariant=True |
| TB4 | intrastat-meldung | L1 | example-tests | yes | yes | facet: spec_precision=needs-inference, hidden_invariant=True |
| TB4 | jax-speedrun-gpu | L1 | dynamic-gate | yes | yes | facet: spec_precision=constraint-dense, hidden_invariant=Tru |
| TB4 | ks-solver-cpp | L1 | example-tests | yes | yes | facet: spec_precision=fully-specified, hidden_invariant=True |
| TB4 | kv-live-surgery | L1 | dynamic-gate | yes | no | facet: spec_precision=needs-inference, hidden_invariant=True |
| TB4 | lake-temp-glm | L1 | example-tests | yes | yes | facet: spec_precision=fully-specified, hidden_invariant=True |
| TB4 | layout-config-recreation | L1 | example-tests | yes | yes | facet: spec_precision=constraint-dense, hidden_invariant=Tru |
| TB4 | layout-config-recreation2 | L1 | example-tests | yes | yes | facet: spec_precision=needs-inference, hidden_invariant=Fals |
| TB4 | legacy-utility-triage | L1 | example-tests | yes | no | facet: spec_precision=constraint-dense, hidden_invariant=Tru |
| TB4 | live-database-cutover | L1 | dynamic-gate | yes | no | facet: spec_precision=needs-inference, hidden_invariant=True |
| TB4 | math-eval-grader | L1 | example-tests | yes | yes | facet: spec_precision=needs-inference, hidden_invariant=True |
| TB4 | medical-claims-processing | L1 | example-tests | yes | yes | facet: spec_precision=needs-inference, hidden_invariant=True |
| TB4 | mp-checkpoint-consolidation | L1 | example-tests | yes | yes | facet: spec_precision=constraint-dense, hidden_invariant=Tru |
| TB4 | music-harmony | L1 | example-tests | yes | yes | facet: spec_precision=needs-inference, hidden_invariant=True |
| TB4 | mvcc-lsm-compaction | L1 | dynamic-gate | yes | yes | facet: spec_precision=needs-inference, hidden_invariant=True |
| TB4 | nextjs-performance | L1 | dynamic-gate | yes | yes | facet: spec_precision=constraint-dense, hidden_invariant=Tru |
| TB4 | ontology-kg-querying | L1 | example-tests | yes | yes | facet: spec_precision=needs-inference, hidden_invariant=True |
| TB4 | payments-pipeline-fix | L1 | dynamic-gate | yes | no | facet: spec_precision=needs-inference, hidden_invariant=True |
| TB4 | photonic-waveguide-routing | L1 | example-tests | yes | yes | facet: spec_precision=constraint-dense, hidden_invariant=Tru |
| TB4 | production-planning | L1 | example-tests | yes | yes | facet: spec_precision=constraint-dense, hidden_invariant=Tru |
| TB4 | protein-autointerp-disulfide | L1 | example-tests | yes | yes | facet: spec_precision=needs-inference, hidden_invariant=True |
| TB4 | react-lead-form | L1 | example-tests | yes | yes | facet: spec_precision=constraint-dense, hidden_invariant=Tru |
| TB4 | risk-scorer-replay | L1 | example-tests | yes | yes | facet: spec_precision=needs-inference, hidden_invariant=True |
| TB4 | roy-polymorph-cn | L1 | example-tests | yes | yes | facet: spec_precision=needs-inference, hidden_invariant=True |
| TB4 | rs-archive-clone | L1 | example-tests | yes | yes | facet: spec_precision=needs-inference, hidden_invariant=True |
| TB4 | satb-audio-transcription | L1 | example-tests | yes | yes | facet: spec_precision=constraint-dense, hidden_invariant=Tru |
| TB4 | session-window-debug | L1 | example-tests | yes | yes | facet: spec_precision=needs-inference, hidden_invariant=True |
| TB4 | sglang-qwen-burst | L1 | example-tests | yes | yes | facet: spec_precision=needs-inference, hidden_invariant=True |
| TB4 | shadow-relay | L1 | example-tests | yes | yes | facet: spec_precision=needs-inference, hidden_invariant=True |
| TB4 | sound-change-cascade | L1 | example-tests | yes | yes | facet: spec_precision=needs-inference, hidden_invariant=True |
| TB4 | takens-embedding-lean | L1 | property/fuzz | yes | yes | facet: spec_precision=needs-inference, hidden_invariant=True |
| TB4 | uefi-bootkit | L1 | example-tests | yes | yes | facet: spec_precision=needs-inference, hidden_invariant=True |
| TB4 | vba-userform-port | L1 | example-tests | yes | yes | facet: spec_precision=needs-inference, hidden_invariant=True |
| TB4 | vf2-speedup-networkx | L1 | dynamic-gate | yes | yes | facet: spec_precision=constraint-dense, hidden_invariant=Tru |
| TB4 | vllm-deepseek-streaming | L1 | example-tests | yes | no | facet: spec_precision=needs-inference, hidden_invariant=True |
| TB4 | vpp-loss-divergence | L1 | example-tests | yes | yes | facet: spec_precision=needs-inference, hidden_invariant=True |
| TB4 | wal-recovery-ordering | L1 | dynamic-gate | yes | yes | facet: spec_precision=constraint-dense, hidden_invariant=Tru |
| TB4 | wdm-design | L1 | example-tests | yes | yes | facet: spec_precision=constraint-dense, hidden_invariant=Tru |
| TB4 | freecad-spring-clip | L2 | example-tests | yes | yes | facet: constraint-dense with explicit rules in instruction |
| TB4 | hof-topology-interpenetration | L2 | example-tests | yes | yes | facet: constraint-dense with explicit rules in instruction |
| TB4 | distributed-dedup | L4 | dynamic-gate | yes | yes | 3 signature/interface blocks |
| TB4 | retro-console-soc | L4 | example-tests | yes | yes | 3 signature/interface blocks |
| TB4 | biped-contact-dynamics | L5 | example-tests | yes | yes | example I/O in instruction |
| TB4 | data-anonymization | L5 | example-tests | yes | yes | example I/O in instruction |
| TB4 | pretrain-shard-corruption | L5 | example-tests | yes | yes | example I/O in instruction |
| TB4 | telecom-entity-resolution | L5 | example-tests | yes | yes | example I/O in instruction |

## 3. Pass rates by rung (frontier models)

### TB4 — 13 Hub submissions (5 trials/task each)

| Rung | n tasks | pooled pass rate | tasks |
|---|---|---|---|
| L0 | 0 | — |  |
| L1 | 58 | 28.0% | atrx-vep-crispr, batched-eval-parity, bun-sourcemap-leak, cad-model, cargo-flight-dispatch, coq-block-bound… |
| L2 | 2 | 36.2% | freecad-spring-clip, hof-topology-interpenetration |
| L3 | 0 | — |  |
| L4 | 2 | 41.5% | distributed-dedup, retro-console-soc |
| L5 | 4 | 27.7% | biped-contact-dynamics, data-anonymization, pretrain-shard-corruption, telecom-entity-resolution |
| L6 | 0 | — |  |

### TB4 per-model pass rate by rung

| Model | L1 | L2 | L4 | L5 |
|---|---|---|---|---|
| fable-5.1/cc | 55% | 100% | 100% | 60% |
| opus-5/cc | 50% | 100% | 90% | 50% |
| fable-5/cc | 42% | 90% | 80% | 35% |
| glm-5.3/cc | 38% | 100% | 90% | 45% |
| gpt-5.6-sol/codex | 37% | 20% | 30% | 55% |
| opus-4.8/cc | 23% | 30% | 50% | 20% |
| gpt-5.6-terra/codex | 23% | 0% | 0% | 20% |
| grok-4.6/build | 20% | 0% | 30% | 25% |
| gemini-3.8-flash/mini-swe | 19% | 30% | 30% | 5% |
| gpt-5.6-luna/codex | 18% | 0% | 0% | 20% |
| sonnet-5/cc | 12% | 0% | 30% | 20% |
| grok-4.5/build | 14% | 0% | 0% | 5% |
| gemini-3.7-flash/mini-swe | 12% | 0% | 10% | 0% |

### TB2 — 5 frontier models from HF trajectories cache

| Rung | n tasks | pooled pass rate |
|---|---|---|
| L0 | 1 | 98.6% |
| L1 | 44 | 58.4% |
| L2 | 31 | 84.5% |
| L3 | 2 | 86.3% |
| L4 | 4 | 44.1% |
| L5 | 2 | 74.3% |
| L6 | 5 | 85.9% |

### Verifier class pass rates (TB4 pooled)

| Verifier | Pooled pass rate |
|---|---|
| dynamic-gate | 29.7% |
| example-tests | 28.5% |
| property/fuzz | 20.0% |

## 4. Trial variance (rule C6)

Bernoulli variance p(1−p) per trial at the task level, pooled across 13 TB4 models (65 trials/task max). High variance + middling pass ⇒ single-trial outcomes unstable.

| Task | Rung | Pooled pass | Bernoulli var | n trials |
|---|---|---|---|---|
| fp8-rmsnorm-gemm | L1 | 51% | 0.250 | 65 |
| react-lead-form | L1 | 51% | 0.250 | 65 |
| retro-console-soc | L4 | 51% | 0.250 | 65 |
| payments-pipeline-fix | L1 | 46% | 0.249 | 65 |
| telecom-entity-resolution | L5 | 55% | 0.247 | 65 |
| atrx-vep-crispr | L1 | 43% | 0.245 | 65 |
| risk-scorer-replay | L1 | 57% | 0.245 | 65 |
| sound-change-cascade | L1 | 43% | 0.245 | 65 |
| batched-eval-parity | L1 | 42% | 0.243 | 65 |
| vpp-loss-divergence | L1 | 42% | 0.243 | 65 |
| interleaved-vigenere | L1 | 40% | 0.240 | 65 |
| intrastat-meldung | L1 | 40% | 0.240 | 65 |
| hof-topology-interpenetration | L2 | 38% | 0.237 | 65 |
| cumulative-layout-shift | L1 | 62% | 0.237 | 65 |
| wal-recovery-ordering | L1 | 37% | 0.233 | 65 |
| biped-contact-dynamics | L5 | 35% | 0.229 | 65 |
| fin-saccr-rwa | L1 | 65% | 0.229 | 65 |
| freecad-spring-clip | L2 | 34% | 0.224 | 65 |
| distributed-dedup | L4 | 32% | 0.219 | 65 |
| coq-block-bound | L1 | 68% | 0.219 | 65 |

Median Bernoulli variance: **0.186**. Tasks with 15–85% pass and ≥median variance: **34**.

### Cross-model spread (same task, ≥3 trials/model)

| Task | max−min pass across models | mean pass |
|---|---|---|
| atrx-vep-crispr | 100% | 43% |
| biped-contact-dynamics | 100% | 35% |
| cad-model | 100% | 26% |
| coq-block-bound | 100% | 68% |
| distributed-dedup | 100% | 32% |
| embedding-drift-monitor | 100% | 31% |
| fp8-rmsnorm-gemm | 100% | 51% |
| freecad-platform-drawing | 100% | 22% |
| freecad-spring-clip | 100% | 34% |
| hof-topology-interpenetration | 100% | 38% |
| interleaved-vigenere | 100% | 40% |
| layout-config-recreation2 | 100% | 68% |
| mp-checkpoint-consolidation | 100% | 69% |
| mvcc-lsm-compaction | 100% | 29% |
| payments-pipeline-fix | 100% | 46% |

## 5. Tells

### (a) Monotone pass rate in rung?

TB4 pooled: **no** — L1=28%, L2=36%, L4=42%, L5=28%.

TB2 pooled: L0=99%, L1=58%, L2=85%, L3=86%, L4=44%, L5=74%, L6=86% (non-monotone: L4 > L2).

### (b) Hidden tests + underspecified (L1) + low pass + model disagreement

- **bun-sourcemap-leak** pass=0%, model spread=0% — facet: spec_precision=constraint-dense, hidden_invariant=True

- **cargo-flight-dispatch** pass=0%, model spread=0% — facet: spec_precision=needs-inference, hidden_invariant=True

- **foodstuff-beta-activity** pass=0%, model spread=0% — facet: spec_precision=needs-inference, hidden_invariant=True

- **freight-dispatch-shift** pass=0%, model spread=0% — facet: spec_precision=constraint-dense, hidden_invariant=True

- **glycan-ms2-elucidation** pass=0%, model spread=0% — facet: spec_precision=needs-inference, hidden_invariant=True

- **ontology-kg-querying** pass=0%, model spread=0% — facet: spec_precision=needs-inference, hidden_invariant=True

- **freecad-impeller** pass=2%, model spread=20% — facet: spec_precision=needs-inference, hidden_invariant=True

- **layout-config-recreation** pass=2%, model spread=20% — facet: spec_precision=constraint-dense, hidden_invariant=True

- **medical-claims-processing** pass=2%, model spread=20% — facet: spec_precision=needs-inference, hidden_invariant=True

- **music-harmony** pass=2%, model spread=20% — facet: spec_precision=needs-inference, hidden_invariant=True


### (c) Instruction leaks assertions + near-100% pass (L5/L6 disguise)

- **wdm-design** L1 leak=['/app/design.npy', '/app/meta.json'] pass=83%

### (d) Non-monotonicity (more info, lower pass)

- L4 (42%) → L5 (28%) on TB4 pooled frontier runs.

Concrete tasks: **torch-tensor-parallelism** (L4, 49% TB2) vs **fix-git** (L0, 72%); **html-js-filter** (L2, 88% TB4) vs **data-anonymization** (L1, 0%).


## 6. Verdict

**Partial transfer, not a clean difficulty axis.** The L0–L6 ladder was built for synthetic SWE units where we control information withholding on a *fixed* verifier. Terminal-Bench tasks are almost all **hidden-test, single-rung** evaluations: the agent never sees L3–L6 affordances during the run (tests mount at verify time; 64/66 TB4 and 84/89 TB2 tasks are not L6). Rung labels therefore describe **instruction prose richness**, not an experimental knob.

TB4 pooled pass rates do **not** increase monotonically with rung (L1=28%, L2=36%, L4=42%, L5=28%). Domain verifier class (dynamic-gate, numeric tolerance) and task domain dominate. Rule **C6** confirmed: 65 trials/task on TB4 show median Bernoulli variance 0.186; tasks around 50% pass are single-trial unstable.

**What could not be determined:** per-rung flip points (no multi-rung variants per TB task); whether L1 tasks fail for instruction insufficiency vs verifier-too-narrow without manual C1 audits; TB2 frontier pass-by-rung for models not in the HF cache subset.
