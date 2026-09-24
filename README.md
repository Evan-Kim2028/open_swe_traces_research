# LadderBench and the Information Ladder

Code behind
[*The Information Ladder: Measuring Model Capabilities*](https://evan-kim2028.github.io/evan_writings/writings/difficulty-is-an-information-gap/).

A coding task is hard because of what its prompt leaves out. The **information ladder** gives one
task 6 prompts, each containing everything in the one below, from a bug report that names only the
symptom to the hidden tests themselves. The first level a model passes measures how much information
it needed. A task that fails from the bug report but passes from the full description is
**certified**: shown to be hard for that model and solvable.

**LadderBench** is the dataset this repository builds: 591 synthetic Go tasks cut from 9 open-source
repositories, 411 of them graded over 1,608 trials in [Harbor](https://github.com/harbor-framework/harbor),
and 242 certified.

## Results

- **The prompt decides whether a task is solvable.** 59% of graded tasks fail from a bug report and
  become solvable with more information, 81% of them once the model gets the full description.
- **Solve rates hide real differences between models.** On 32 of 73 shared tasks, Composer 2.5 and
  SWE-2 need different amounts of information, and on tasks with long descriptions they agree only
  25% of the time.
- **Information replaces search.** When extra information turns a failure into a pass, the model
  makes 24–32% fewer reads and searches.

## The ladder

| Post | Code | The solver gets |
|---|---|---|
| L1 | `L0` | Bug report: the symptom and a command to reproduce it |
| L2 | `L2` | Full description: every behavior the hidden tests check |
| L3 | `L3` | L2 plus the hidden test names |
| L4 | `L4` | L3 plus the exported signatures as stubs |
| L5 | `L5` | L4 plus one hidden test file in the repository |
| L6 | `L6` | Every hidden test in the repository |

The code numbers the bug report `L0` and keeps a retired `L1`. Rung keys are baked into job
directories and ledger rows, so the renumbering is tracked in
[`docs/tech_debt_level_numbering.md`](docs/tech_debt_level_numbering.md).

## Where each part of the post lives

| Post section | Code |
|---|---|
| Six levels of information | [`synth/affordance.py`](src/openswe_traces/synth/affordance.py) adds information level by level to a Harbor task |
| Task certification | [`ladder/ledger.py`](src/openswe_traces/ladder/ledger.py) reads every trial outcome; [`ladder/escalate.py`](src/openswe_traces/ladder/escalate.py) climbs a task that fails L1 and L2 |
| Agentic task generation | [`pipeline/`](src/openswe_traces/pipeline/) and [`authoring/`](src/openswe_traces/authoring/); stages and information barriers in [`analytics/research/PIPELINE.md`](analytics/research/PIPELINE.md) |
| Validation checks | [`synth/rules.py`](src/openswe_traces/synth/rules.py), one verdict per rule in [`analytics/research/verifier_rules.md`](analytics/research/verifier_rules.md) |
| Trial integrity | [`pipeline_ext/hack_audit.py`](src/openswe_traces/pipeline_ext/hack_audit.py) audits every pass; [`analysis/harness_audit.py`](src/openswe_traces/analysis/harness_audit.py) checks each verdict measured the model |
| Cost | [`reports/paper_numbers.py`](src/openswe_traces/reports/paper_numbers.py), [`reports/devin_usage.py`](src/openswe_traces/reports/devin_usage.py), [`reports/token_cost.py`](src/openswe_traces/reports/token_cost.py) |
| Information replaces search | [`analysis/traces.py`](src/openswe_traces/analysis/traces.py) |
| Comparing model capabilities | [`analysis/disagreement.py`](src/openswe_traces/analysis/disagreement.py) |

## Reproducing the numbers

Every count in the post comes from one module:

```bash
uv sync
uv run python -m openswe_traces.reports.paper_numbers
```

The charts are drawn by
[`scripts/information-gap-figures.py`](https://github.com/Evan-Kim2028/evan_writings/blob/main/scripts/information-gap-figures.py)
in the blog repository, which reads the ledger in this one.

Both need the trial records under `experiments/dose_response/`. That directory holds the staged
Harbor tasks and every trial, about 63 GB, and is not in git. This repository holds the code, the
pilot task sets under `experiments/harbor_nex/` and `experiments/pipeline/`, and the research notes.

## Repository map

| Path | What is there |
|---|---|
| `src/openswe_traces/ladder/` | Trial outcomes, certificates, admission rules, escalation |
| `src/openswe_traces/pipeline/`, `authoring/`, `pipeline_ext/` | The task factory and its audits |
| `src/openswe_traces/synth/` | Harbor task packaging, the ladder levels, validation rules |
| `src/openswe_traces/reports/` | Status, cost, and the post's numbers |
| `src/openswe_traces/analysis/` | Instruments behind the findings in `analytics/research/` |
| `src/openswe_traces/ops/` | Agent connectors, capacity, budgets, container hygiene |
| `scripts/ops/` | Commands and daemons that run the experiment ([README](scripts/ops/README.md)) |
| `experiments/` | Pilot task packages; staged sweeps and trials (gitignored) |
| `analytics/research/` | Findings journal, one note per question |
| `docs/` | Architecture, operations, specs, handoffs |
| `tests/` | pytest |

## Documentation

- [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md): vocabulary, trial lifecycle, and module layout. Read first.
- [`docs/OPERATIONS.md`](docs/OPERATIONS.md): how to run the experiment, and the rules learned running it.
- [`analytics/research/PIPELINE.md`](analytics/research/PIPELINE.md): the task factory as it runs.
- [`analytics/research/verifier_rules.md`](analytics/research/verifier_rules.md): every validation rule and what it caught.
- [`docs/HANDOFF.md`](docs/HANDOFF.md): the latest state of the run.

## Setup

```bash
uv sync
uv run pytest
```

Running trials needs [Harbor](https://github.com/harbor-framework/harbor), Docker, and API access
for the agents under test. See [`docs/OPERATIONS.md`](docs/OPERATIONS.md).

## Earlier work

This repository started as local analytics on
[nvidia/Open-SWE-Traces](https://huggingface.co/datasets/nvidia/Open-SWE-Traces). That work is
documented in [`docs/open_swe_traces_analytics.md`](docs/open_swe_traces_analytics.md), with notes
in [`notes/`](notes/README.md).

## License

Code: MIT. Open-SWE-Traces data: [CC BY 4.0](https://huggingface.co/datasets/nvidia/Open-SWE-Traces).
