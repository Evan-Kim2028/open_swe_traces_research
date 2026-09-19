# Pipeline repo set (Go)

Generated 2026-09-19 02:14 UTC by `uv run python scripts/prepare_pipeline_repos.py`.

Source: `outputs/task_difficulty.parquet` mid+hard instance counts among helm/helm, argoproj/argo, knative/client, google/go-github, kubernetes/kops, goadesign/goa, nats-io/nats-server. Excluded: tikv/client-go, vaskoz/dailycodingproblem-go, mgechev/revive.

Identity pass: module path + brand strings only (`obfuscate.py --identity-only`); no symbol or directory renames. Images `ladder-base:<name>` from golang:1.23, `go mod download`, `go build ./...`. Baseline: `docker run --network none go test ./... -count=1 -timeout 20m`. Keep if build succeeds and ≥5 packages pass.

## Kept

| name | url | commit | base_image | packages_passing | notes |
|---|---|---|---|---:|---|
| *(none)* | | | | | |

## Dropped

| name | reason |
|---|---|
| `knative-client` | offline tests: 0 packages passed (need >=5); fail=0 skip=0 |
| `goa` | offline tests: 0 packages passed (need >=5); fail=0 skip=0 |
| `go-github` | offline tests: 0 packages passed (need >=5); fail=0 skip=0 |
| `kops` | offline tests: 0 packages passed (need >=5); fail=0 skip=0 |
| `argo` | image build failed (go build ./... or earlier Dockerfile step) |
| `nats-server` | offline tests: 0 packages passed (need >=5); fail=0 skip=0 |
| `helm` | offline tests: 0 packages passed (need >=5); fail=0 skip=0 |

## Reproduce

```
uv run python scripts/prepare_pipeline_repos.py
```

At most 2 image builds at a time, `nice -n 15`. Does not touch `experiments/harbor_nex/jobs`.
