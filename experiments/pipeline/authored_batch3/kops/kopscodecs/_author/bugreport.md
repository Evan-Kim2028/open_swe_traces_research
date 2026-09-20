# Bug report

Reading and writing cluster objects is broken: documents decode to the wrong type or fail outright, objects serialize to empty output, and round-tripping a cluster spec loses its fields. Unrecognized media types are silently accepted instead of rejected.

Expected: a `Cluster` serializes to YAML/JSON carrying its apiVersion and kind; a document with a known apiVersion decodes to the typed object; a document with an unrelated apiVersion decodes to an unstructured object; an unsupported media type is an error.

Reproduce with:

```
go test -count=1 ./pkg/kopscodecs/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository.
