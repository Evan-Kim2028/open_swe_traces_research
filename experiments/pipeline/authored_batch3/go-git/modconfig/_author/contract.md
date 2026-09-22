# Contract — modconfig

Hidden suite: `tests/hidden/config/modconfig_bb_test.go`
(package `config`, in-package). One `TestDetailNN` per DETAILS.md line.

| test | detail | inferable | asserts |
|---|---|---|---|
| TestDetail01 | 1 | doc | `validSubmoduleName` rejects: `""`, `"."`, NUL byte, leading/trailing `/` and `\`, `x:foo`, `x:y` (drive-letter), `a/../b`, `a\..\b`, `a/.. /b` (NTFS-folded), `a/.<U+200C>./b` (HFS+-folded); accepts `a/b/c`, `a/b..c`, `sub`. |
| TestDetail02 | 2 | no | SHAPE: a `[submodule "a/../b"]` entry never reaches `Submodules` while a good one does; empty `path`/`url` entries still land in the map — they do NOT skip. |
| TestDetail03 | 3 | partially | `Validate` → `ErrModuleBadPath` for `a/../b`, `a\..\b`, `../x`, `x/..`, `x/../`; legal for `a/x../b`, `a/..x/b`, `plain`. |
| TestDetail04 | 4 | no | SHAPE: marshaling a submodule with empty Name still emits a `submodule` section carrying its path text — the path supplies the label. |
| TestDetail05 | 5 | no | SHAPE: a `Branch` with empty remote/merge/rebase/description marshals a subsection with NONE of those four options present (`HasOption` false). |
| TestDetail06 | 6 | doc | `Rebase` accepts `true`, `interactive`, `false`; `"sometimes"` → `errBranchInvalidRebase`; `Merge: "plainname"` (no `refs/`) validates. |
| TestDetail07 | 7 | doc | `Description: "line1\nline2"` marshals with no raw newline but a literal `\n` escape; unmarshal restores the real newline. |
| TestDetail08 | 8 | doc | `parseConfigBool`: `true/TRUE/yes/on/1/-1/42` → `OptBoolTrue`; `false/NO/off/0` → `OptBoolFalse`; `""` → `OptBoolUnset`. |
| TestDetail09 | 9 | doc | `parseConfigBool("maybe"/"2x"/"tru")` → `OptBoolUnset`. |
| TestDetail10 | 10 | no | SHAPE: `OptBoolUnset.String()` is non-empty and differs from both the true and false spellings. |
| TestDetail11 | 11 | no | SHAPE: after Unmarshal+Marshal, a raw `unknownkey = keepme` option survives — the raw subsection is preserved. |
| TestDetail12 | 12 | no | SHAPE: `Validate` on all-bad → `ErrModuleBadName` first; missing URL → `ErrModuleEmptyURL`; `..`-path → `ErrModuleBadPath` — distinct sentinels identify check order. |

Refusals/softening: line 2's skip-vs-error distinction is asserted at the observable level
(bad entries absent from the map, empty-field entries present) rather than the internal
mechanism; line 12 pins order via which sentinel surfaces, not error text.
