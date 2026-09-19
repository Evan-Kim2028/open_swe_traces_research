# pipeline repos2

Date: 2026-09-18. Extra Go hosts (nats-io/nats-server, spf13/cobra,
gin-gonic/gin) cloned at HEAD, `.git` stripped, identity-obfuscated
(module path + brand strings only; no symbol/dir renames), then built as
`ladder-base:<name>` (`golang:1.23` + tree + `go mod download` + `go build ./...`).
Offline `go test ./... -count=1 -timeout 20m` with `--network none`.
Keep if the image builds and ≥ 5 packages pass.
Image builds ran one at a time under `nice -n 10`.
Internet: `git clone`, `go mod download`, and `GOTOOLCHAIN=auto` (gin and
nats-server declare go 1.26; the image is still `FROM golang:1.23`).

Reproduce:

```
uv run python scripts/prepare_repos2.py
```

Output: `experiments/pipeline/repos2/<name>/` (`src/`, `Dockerfile`,
`baseline.json`), plus `experiments/pipeline/repos2.yaml`.

| repo | HEAD | image | build | pkgs pass/fail/skip | test wall | keep | notes |
|---|---|---|---|---:|---:|---|---|
| `nats-io/nats-server` | `8ad52657d1f3` | `ladder-base:nats-server` | yes | 10/3/8 | 1239.3s | yes | kept; server package hit 20m timeout; GOTOOLCHAIN=auto fetched go1.26.8 |
| `spf13/cobra` | `adbc8813901b` | `ladder-base:cobra` | yes | 1/1/0 | 2.8s | no | build ok; only 2 packages exist, 1 passed offline (need 5) |
| `gin-gonic/gin` | `5c6a15f8f956` | `ladder-base:gin` | yes | 5/1/1 | 11.1s | yes | kept; root package failed 3 tests; GOTOOLCHAIN=auto fetched go1.26 |

Identity (module path + brand strings only):

| repo | old module | new module |
|---|---|---|
| `nats-io/nats-server` | `github.com/nats-io/nats-server/v2` | `example.internal/msgbus/v2` |
| `spf13/cobra` | `github.com/spf13/cobra` | `example.internal/clikit` |
| `gin-gonic/gin` | `github.com/gin-gonic/gin` | `example.internal/httprouter` |

