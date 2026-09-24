# Details — tfhcl2

1. `object.Write` emits `key {\n ... }` with fields sorted by key; consecutive single-value fields get `=` aligned over the longest key in that RUN — a non-single-value field resets the alignment. Inferable: no — run-based alignment is fiddly.
2. `toElement` maps bool→`true/false`, ints→decimal, string→quoted literal, `*Literal`→itself (nil→omit), nil pointers→omit, struct→object of visible fields skipping nil elements, slices→slice path; unsupported kinds panic. Inferable: partially — the kind coverage is a choice.
3. Empty slice → nil element (the field is omitted entirely, no `key = []`). Inferable: no — omit-vs-empty-block.
4. `[]*Literal` and `[]string`/`[]*string` collapse to a single `[a, b]` list expression; `[]struct` stays a multi-block `sliceObject` where each member writes under the same key. Inferable: no — which slice kinds collapse is arbitrary.
5. `fieldKey` prefers the `cty` tag, else snake_cases the Go name inserting `_` only where an uppercase follows a lowercase — `MyIPAddr` → `my_ipaddr`, `HTTPPort` → `http_port`. Inferable: no — acronym handling is a specific choice.
6. `mapStringLiteral.Write` emits nothing at all for an empty map; otherwise `key = {\n` + sorted, aligned `"k" = v` lines + `}`; alignment pads to the longest QUOTED key. Inferable: partially — empty-suppression is observable in the doc example shape.
7. `mapToElement` accepts `map[string]string` (values quoted) and `map[string]*Literal` (values verbatim); other key/value kinds panic. Inferable: no.
8. `quote` escapes only `"` and `\` — newlines and tabs pass through unescaped. Inferable: no — minimal escape set.
9. `writeIndent` emits `indent` single spaces; callers indent two levels per nesting. Inferable: partially.
10. `finishHCL2` section order: locals+outputs, provider blocks, resources, data sources, then the `terraform {}` requirements block; the result lands in `Files["kubernetes.tf"]`. Inferable: partially — section order is a choice.
11. `writeLocalsOutputs`: scalar outputs go to the `locals` object, array outputs collapse to a `[...]` list expression there, then one `output "name"` block per name in sorted order. Inferable: no.
12. Provider naming: GCE→`google`, Hetzner→`hcloud`, Azure→`azurerm`, others verbatim; `project` only for GCE, `region` for all except Hetzner/DO/Azure, `zone` for Scaleway, `subscription_id` for Azure when set; azurerm gets an empty `features` block; extra providers each carry `alias = "files"`. Inferable: no — per-provider body rules.
13. Resources and data sources iterate types then names, both sorted, under `resource "t" "n"` / `data "t" "n"` headers separated by blank lines. Inferable: partially.
14. The `terraform` block pins `required_version = ">= 0.15.0"` and one required_providers entry per provider (sorted), with a fixed source/version table; an unlisted provider aborts (klog.Fatalf); aliased providers also emit `configuration_aliases = [provider.alias]`. Inferable: no — version pins are arbitrary literals.
