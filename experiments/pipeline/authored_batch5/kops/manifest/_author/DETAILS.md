# Details — manifest

1. `LoadObjectsFrom` skips sections whose only lines are empty or `#`-comments —
   `hasYAMLContent` requires a non-empty non-comment line. Inferable: yes — empty sections
   would otherwise break yaml.Unmarshal.
2. `ObjectList.ToYAML` joins with `\n---\n\n` and drops `IsEmptyObject` entries.
   Inferable: no — separator bytes are arbitrary.
3. Field getters return `""` for missing OR wrong-typed fields — no error is raised.
   `GetName`/`GetNamespace` read `metadata.name`/`metadata.namespace`. Inferable: yes.
4. `Reparse` navigates intermediate fields as `map[string]interface{}` — a missing field or
   non-map value is an error naming the field and the full dotted path. Inferable:
   partially — error text arbitrary.
5. `Object.Set` requires all intermediate path segments to already exist as maps — it does
   NOT create missing intermediates. The leaf is set to a yaml-round-tripped map copy of
   `newValue` (scalars become... whatever yaml produces — callers pass maps). Inferable:
   partially — no-create is a real semantic.
6. `visit` mutates in place: mutators write back into the parent map/slice; path elements
   are dotted field names and `[i]` for slice indexes. `[]string` is silently skipped;
   other concrete types error. Inferable: partially — the []string hole is odd.
7. `visitorBase` callbacks are no-ops (log only) — real visitors embed it and override.
   Inferable: yes.
8. `accept` fatal-logs if a mutator tries to replace the root — top-level data can't be
   replaced. Inferable: partially.
