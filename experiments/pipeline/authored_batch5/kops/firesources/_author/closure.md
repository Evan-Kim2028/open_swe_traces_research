# Closure — firesources

Package: `upup/pkg/fi` (`example.internal/clustkit/upup/pkg/fi`).

Files: `upup/pkg/fi/resources.go` (19 funcs).

Removed functions (bodies stubbed): `ResourcesMatch`, `CopyResource`, `ResourceAsString`,
`ResourceAsBytes`, `StringResource.MarshalJSON`, `NewStringResource`, `StringResource.Open`,
`BytesResource.MarshalJSON`, `NewBytesResource`, `BytesResource.Open`, `NewFileResource`,
`FileResource.Open`, `NewVFSResource`, `VFSResource.Open`, `TaskDependentResource.Open`,
`TaskDependentResource.GetDependencies`, `TaskDependentResource.IsReady`, `FunctionToResource`,
`functionResource.Open`.

Exported entry point(s): `ResourcesMatch` is the change-detection primitive for every `fi`
task's `Find`; the resource constructors back all template/file rendering.

Test files removed in excision: `dryruntarget_test.go`, `files_test.go`.
