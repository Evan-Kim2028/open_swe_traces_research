# Exported API — kopscodecs

Package `pkg/kopscodecs` (importable as `example.internal/clustkit/pkg/kopscodecs`).

- `func ToVersionedYaml(obj runtime.Object) ([]byte, error)` — encode to YAML in the default API version.
- `func ToVersionedJSON(obj runtime.Object) ([]byte, error)` — encode to JSON in the default API version.
- `func ToVersionedYamlWithVersion(obj runtime.Object, version runtime.GroupVersioner) ([]byte, error)`
- `func ToVersionedJSONWithVersion(obj runtime.Object, version runtime.GroupVersioner) ([]byte, error)`
- `func ToMediaTypeWithVersion(obj runtime.Object, mediaType string, gv runtime.GroupVersioner) ([]byte, error)` — encode to an arbitrary supported media type.
- `func Decode(data []byte, defaultReadVersion *schema.GroupVersionKind) (runtime.Object, *schema.GroupVersionKind, error)` — decode YAML/JSON, returning the object and its GroupVersionKind.
- `var Scheme`, `var Codecs`, `var ParameterCodec` — the registered scheme and codec factory (untouched).

Production callers: `pkg/client/simple/vfsclientset/commonvfs.go`, `pkg/model/config.go`, `pkg/testutils/modelharness.go`, `cmd/kops/{get,create,delete}*.go`.
