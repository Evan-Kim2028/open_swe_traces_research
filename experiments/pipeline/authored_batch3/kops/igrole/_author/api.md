# Exported API — igrole

Package `pkg/apis/kops` (importable as `example.internal/clustkit/pkg/apis/kops`).

- `func ParseInstanceGroupRole(input string, lenient bool) (InstanceGroupRole, bool)` — convert a string to an `InstanceGroupRole`; with `lenient` set, pluralised words match too. Returns `(role, true)` on a match.
- `func ParseRawYaml(data []byte, dest interface{}) error` — parse an object with plain yaml, no API machinery (deprecated).
- `func ToRawYaml(obj interface{}) ([]byte, error)` — marshal an object to yaml, no API machinery (deprecated).

Production callers: `pkg/apis/kops/channel.go` (`ParseRawYaml`), `cmd/kops/create_instancegroup.go`, `pkg/testutils/modelharness.go`, `nodeup/pkg/model/kube_apiserver.go`.
