# Sources

## Primary dataset and paper

| ID | Title | Link | Notes |
|---|---|---|---|
| OST-2026 | Open-SWE-Traces: Dual-Mode Multilingual Distillation | https://arxiv.org/abs/2606.16038 | Primary dataset paper; filtering, SFT ablations |
| OST-HF | nvidia/Open-SWE-Traces (HuggingFace) | https://huggingface.co/datasets/nvidia/Open-SWE-Traces | 512k rows, ~43 GB; messages, tools, resolved, patches |
| OST-COL | Open-SWE-Collections | https://huggingface.co/collections/nvidia/open-swe-traces | Fine-tuned Open-SWE-Agent models |

## Trace quality and curation

| ID | Title | Link | Notes |
|---|---|---|---|
| SWE-Prime | SWE-Prime: Fewer Trajectories, Better Performance | https://arxiv.org/abs/2608.27449 | Trajectory + segment quality; 10% subset beats full resolved |
| Curation-2607 | Systematic Evaluation of Trajectory Data Curation for LoRA | https://arxiv.org/abs/2607.17205 | Efficiency + Style axes; error-retry rate; CE loss proxy |
| SRFT | Step Rejection Fine-Tuning | https://arxiv.org/pdf/2605.10674 | Critic step masking vs discard failures |
| SWE-Lego | SWE-Lego: Limits of SFT for Issue Resolving | https://arxiv.org/pdf/2601.01426 | Error masking; semi-resolved recycling; git-hacking filter |

## SFT pipelines and agent training

| ID | Title | Link | Notes |
|---|---|---|---|
| SWE-smith | SWE-smith trajectories + train guide | https://huggingface.co/datasets/SWE-bench/SWE-smith-trajectories · https://swesmith.com/guides/train_swe_agent/ | Resolved-only; XML format; Claude 3.7 trajectories |
| SWE-Master | SWE-Master: Post-Training for SWE Agents | https://arxiv.org/abs/2602.03411 | Env response masking; SFT + RL |
| SWE-Master-code | SWE-Master repository | https://github.com/RUCAIBox/SWE-Master/ | Pre-tokenize + loss mask scripts |
| SWE-Hero | SWE-Zero to SWE-Hero | https://arxiv.org/html/2604.01496 | Execution-free → execution-based; tool output masking |
| TuFT-SFT | Chat SFT / assistant-only loss | https://agentscope-ai.github.io/TuFT/en/latest/user-guide/chat-sft.html | Loss masking mechanics |

## Environments and verifiers

| ID | Title | Link | Notes |
|---|---|---|---|
| SWE-Gym | Training Agents and Verifiers with SWE-Gym | https://arxiv.org/pdf/2412.21139v2.pdf | Executable envs; verifier RM; trajectory labeling |
| R2E-Gym | R2E-Gym paper + repo | https://r2e-gym.github.io/ · https://github.com/R2E-Gym/R2E-Gym | Procedural envs; hybrid verifiers; SFT trajectories |
| R2E-SFT | R2EGym-SFT-Trajectories | https://huggingface.co/datasets/R2E-Gym/R2EGym-SFT-Trajectories | ~3k successful trajectories |

## External union candidates

| ID | Dataset | Link |
|---|---|---|
| Pearson | pearsonkyle/swe-agentic-trajectories | https://huggingface.co/datasets/pearsonkyle/swe-agentic-trajectories |
| Nebius | nebius/SWE-agent-trajectories | https://huggingface.co/datasets/nebius/SWE-agent-trajectories |
| Lego-Verified | PrimeIntellect/SWE-Lego-Real-Data-Verified | https://huggingface.co/datasets/PrimeIntellect/SWE-Lego-Real-Data-Verified |
| Lego-Real | SWE-Lego/SWE-Lego-Real-Data | https://huggingface.co/datasets/SWE-Lego/SWE-Lego-Real-Data |
| Review-Traj | SWE-Lego/SWE-Review-Traj | https://huggingface.co/datasets/SWE-Lego/SWE-Review-Traj |

## Tools and references

| ID | Title | Link |
|---|---|---|
| DuckDB-skills | DuckDB Skills for Claude Code | https://duckdb.org/2026/09/16/duckdb-skills |
| SWE-rebench-V2 | Upstream task corpus | https://arxiv.org/abs/2602.23866 (cited in OST paper) |
| Evaltrials | Local index entry | https://github.com/Evan-Kim2028/open_swe_traces_research (see also evaltrials registry in Documents) |

## BibTeX (primary)

```bibtex
@article{ahmad2026openswetraces,
  title={Open-SWE-Traces: Advancing Dual-Mode Multilingual Distillation for Software Engineering Agents},
  author={Wasi Uddin Ahmad and Nikolai Ludwig and Somshubra Majumdar and Boris Ginsburg},
  year={2026},
  eprint={2606.16038},
  archivePrefix={arXiv},
  primaryClass={cs.CL},
  url={https://arxiv.org/abs/2606.16038}
}
```
