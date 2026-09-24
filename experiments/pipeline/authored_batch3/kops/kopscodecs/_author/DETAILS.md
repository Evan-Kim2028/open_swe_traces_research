# Details — kopscodecs

1. `ToVersionedYaml`/`ToVersionedJSON` encode in the default registered version (v1alpha2). Inferable: yes — the version import names it.
2. An unsupported media type is an error mentioning "no serializer". Inferable: partially — error-vs-empty is a choice.
3. `*unstructured.Unstructured` input bypasses version remapping and encodes with the raw media-type serializer; typed objects go through `EncoderForVersion`. Inferable: no — the unstructured bypass is arbitrary.
4. `Decode` first decodes into unstructured to read the GVK, then only re-decodes typed when the group belongs to the clustkit API. Inferable: partially.
5. A document whose apiVersion group is not a clustkit group decodes to the unstructured object as-is, with its GVK returned. Inferable: partially.
6. The legacy short api group is accepted and triggers a byte-level apiVersion rewrite before the typed decode; the current group decodes directly. Inferable: no — which group strings are accepted is hidden in the excised body.
7. The apiVersion rewrite scans line-wise for `apiVersion:` and rewrites only the legacy `v1alpha2` spelling, leaving other versions untouched. Inferable: no.
8. A first-stage decode failure propagates (object, gvk, error) without attempting the typed decode. Inferable: yes.
9. Encode errors are wrapped with the object type and whether the structured or unstructured encoder was used. Inferable: no — wrap shape is arbitrary.
