# Bug report

Terraform output generation produces malformed HCL: block bodies render empty or scrambled, object and map fields lose their ordering and column alignment, string and map keys are not quoted correctly, empty collections still emit blocks (or non-empty ones go missing), struct fields are not renamed to snake_case, and the generated file is missing whole sections.

Expected: a struct field `InstanceID` renders as `instance_id`; a map emits sorted, aligned `"key" = value` lines inside `key = { ... }`; an empty map emits nothing; outputs land in a `locals` block plus sorted `output "name"` blocks; resources and data sources each render in sorted order under `resource "type" "name"` / `data "type" "name"` headers.

Reproduce with:

```
go test -count=1 ./upup/pkg/fi/cloudup/terraform/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository.
