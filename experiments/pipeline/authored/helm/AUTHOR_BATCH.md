# AUTHOR_BATCH — helm (chartkit-obf) feature-excision units, hardest first

Base tree: `experiments/pipeline/repos/helm/src` (identity-obfuscated helm, module `example.internal/chartkit/v4`). Base image `ladder-base:helm`. All units sit in packages whose tests pass OFFLINE per `repos/helm/test.jsonl` (`ok` lines).
Each unit lives at `experiments/pipeline/authored/helm/<unit>/_author/` with `closure.md`, `api.md`, `contract.md` (coverage table over the repo's own tests), `bugreport.md`, `gold.patch`, `cheat.patch`, `excised/excision.patch`, `difficulty.md`. Excision = function bodies replaced by `panic("excised: <name>")` stubs (signatures/doc comments kept, imports blanked where orphaned, compiles clean, fails at runtime). Gold restores exact bodies. Cheat patches make an entry point return canned/simplified output. No tests written.

| # | unit | files | funcs | ~lines excised | hardness driver |
|---|------|-------|-------|---------------:|-----------------|
| 1 | `repindex` | 1 | 10 | ~350 | semver-range resolution to highest match incl. wildcard/hyphen ranges, absolute-vs-relative URL joining, dedup-on-merge, sorted-on-load |
| 2 | `strvalsparser` | 2 | 12 | ~500 | hand-rolled recursive grammar: mode-dependent escapes (normal vs literal vs file), list-index growth, dup-index errors, nested-level cap, type inference only outside string mode |
| 3 | `coalesce` | 1 | 10 | ~370 | asymmetric nil semantics: user-nil erases default but subchart-nil must not shadow global; alias-aware dep traversal; coalesce vs merge modes |
| 4 | `depresolver` | 1 | 4 | ~260 | scheme-dispatched resolution (oci/file/repo/alias), local path digests version from disk not constraint, canonical-sort lock hash |
| 5 | `chartloader` | 3 | 11 | ~330 | 3 entry paths -> one file-list loader; path-traversal/backslash rejection, symlink/device skip, size budget, BOM strip, order-sensitive merge |
| 6 | `memorydriver` | 2 | 11 | ~340 | concurrent state store: version-sorted records, newest-per-name List/Query, replace-in-place, rwlock discipline |
| 7 | `provenance` | 1 | 10 | ~400 | clearsign wire format: sha256 sums of archive (not metadata) in message block, status-object verification, armor/keybox decode |
| 8 | `storage` | 1 | 12 | ~290 | revision history contract: newest-per-name vs history, deployed-status-protected pruning, prune-failure tolerance, corrupt-row skip |
| 9 | `kindsorter` | 3 | 12 | ~370 | kind ordering (unknowns last, alpha among themselves), hook weight+kind sort, stable sort, release list sorts |
| 10 | `ignorerules` | 1 | 6 | ~180 | gitignore-adjacent: no negation, dir-only trailing slash, ** vs *, root anchoring |

## Notes

- Every excision was generated mechanically (`scripts/excise_funcs.py`) and `gofmt`-checked; imports orphaned by stubbing are blank-imported so the tree still compiles. Compile-check against the module graph was not possible offline (deps absent from local module cache) — the verifier image `ladder-base:helm` has them.
- `storage` (pkg/storage) and `memorydriver` (pkg/storage/driver) overlap conceptually but are disjoint closures in different packages; hidden tests for one do not exercise the other's stubbed bodies.
- `kindsorter` keeps the `InstallOrder`/`UninstallOrder` data tables in-tree — only the sort/parse logic is excised.
- `ignorerules` is the batch control (`control: true` in its difficulty.md) — smallest surface, best-documented semantics.
- predicted_flip lines: repindex L4, strvalsparser L4, coalesce L4, depresolver L4, chartloader L3, memorydriver L3, provenance L3, storage L3, kindsorter L2, ignorerules L2.
