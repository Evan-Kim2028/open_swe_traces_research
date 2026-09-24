# Closure — manifest

Package: `pkg/kubemanifest` (`example.internal/clustkit/pkg/kubemanifest`).

Files: `manifest.go` (19 funcs), `visitor.go` (5 funcs) — 24 funcs.

Removed functions (bodies stubbed): `NewObject`, `Object.ToUnstructured`,
`Object.GroupVersionKind`, `FromRuntimeObject`, `LoadObjectsFrom`, `hasYAMLContent`,
`ObjectList.ToYAML`, `Object.ToYAML`, `Object.MarshalJSON`, `Object.accept`,
`Object.IsEmptyObject`, `Object.Kind`, `Object.GetNamespace`, `Object.GetName`,
`getStringValue`, `Object.metadata`, `Object.APIVersion`, `Object.Reparse`, `Object.Set`,
`visitorBase.VisitString`, `visitorBase.VisitBool`, `visitorBase.VisitFloat64`,
`visitorBase.VisitMap`, `visit`.

Exported entry point(s): `LoadObjectsFrom`/`ObjectList.ToYAML` — used by every addon
manifest load and by the image-remap/critical-pod mutators in the same package.

Test files removed in excision: none (kept tests don't reach the closure).
