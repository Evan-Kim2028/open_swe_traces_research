# AUTHOR_BATCH — gin (identity-obfuscated, module `example.internal/httprouter`)

Repo: `experiments/pipeline/repos2/gin/src`. Base image: `ladder-base:gin`.
Eligible packages (offline `ok` in baseline.json): `binding`, `render`, `ginS`,
`internal/bytesconv`, `internal/fs`. Root package excluded (baseline `fail`).
All units: excision stubs bodies with `panic("excised: <name>")`; signatures kept;
unused imports rewritten to `_ "path"` so the excised tree builds. Verified on
host: excision applies + builds (`go build`), gold.patch applies on the excised
tree and restores byte-identical sources, cheat.patch applies and builds
(compiles, returns zero values — fails the suite). No test files touched.

Ranked hardest-first. Closure = files / approx lines / stubbed funcs.

| # | unit | package | closure | predicted_flip |
|---|------|---------|---------|----------------|
| 1 | formmapping | binding | 1 file / 550 / 23 | L5 |
| 2 | bodydecoders | binding | 8 files / ~300 / 30 | L3 |
| 3 | jsonrenders | render | 1 file / 194 / 13 | L3 |
| 4 | validator | binding | 1 file / 98 / 5 | L3 |
| 5 | defaultengine | ginS | 1 file / 159 / 25 | L3 |
| 6 | multipartfiles | binding | 1 file / 74 / 3 | L3 |
| 7 | requestbinders | binding | 4 files / 177 / 14 | L3 |
| 8 | htmlrender | render | 1 file / 107 / 5 | L2 |
| 9 | streamrenders | render | 4 files / ~150 / 10 | L2 |
| 10 | bindingdispatch (control) | binding | 1 file / 127 / 2 | L2 |

## Notes per unit

1. **formmapping** — the struct←key/value reflection engine used by every form,
   query, header, URI and multipart binder. Tag-option parsing (`default=`,
   `parser=`, `collection_format`), lazy pointer allocation doubling as cycle
   protection, per-kind scalar coercion, time/duration formats, JSON fallback
   for struct/map fields, custom `UnmarshalParam`/`TextUnmarshaler` precedence,
   map-target special cases. Largest invariant surface in the repo.
2. **bodydecoders** — eight binder files sharing a decode→validate sequence;
   JSON carries two mutable global decoder flags (UseNumber,
   DisallowUnknownFields), protobuf asserts proto.Message and skips validation,
   plain chases pointers into string/[]byte. Multi-file, stateful.
3. **jsonrenders** — six JSON variants, each a different subtle output rule:
   array-only secure prefix, JS-escaped callback wrapping, per-rune \uXXXX
   ASCII escaping, three distinct content-type strings, encoder path with HTML
   escaping off (trailing newline).
4. **validator** — once-only engine init, pointer re-dispatch by element kind,
   slice/array fan-out with `[i]:`-tagged error aggregation, skip-all-else.
5. **defaultengine** — 25 wrappers around one lazily-created shared engine;
   cross-call state is the real contract (fresh-engine cheat loses Routes()).
6. **multipartfiles** — files-before-values precedence, *FileHeader vs
   FileHeader vs slice vs fixed-array kinds, exact exported errors.
7. **requestbinders** — one engine fed by four sources: full Form (32MB
   multipart cap), PostForm-only, URL query, canonicalized headers, `uri` tag.
8. **htmlrender** — production vs re-parse-per-render debug modes, files > glob
   > fs+patterns source priority ending in panic, name-vs-root execute.
9. **streamrenders** — Reader/Data/Redirect/String boundary rules: >=0 vs >0
   length, fill-only-unset headers, 201 exception inside the 3xx panic gate.
10. **bindingdispatch** — control. 14-row MIME table, GET override, dual
    aliases; mechanical once the table is stated.
