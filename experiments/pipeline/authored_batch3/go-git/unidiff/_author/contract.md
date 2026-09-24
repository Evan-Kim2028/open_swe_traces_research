# Contract — unidiff

Hidden suite: `tests/hidden/plumbing/format/diff/unidiff_bb_test.go`
(package `diff`, in-package). One `TestDetailNN` per DETAILS.md line.
Stubs (`tFile`/`tChunk`/`tFilePatch`/`tPatch`) implement the intact `Patch`
interfaces; output is asserted on the emitted text.

| test | detail | inferable | asserts |
|---|---|---|---|
| TestDetail01 | 1 | partially | First line is `diff --git a/a.txt b/a.txt`; a content change emits `index <h1>..<h2> 100644` (mode suffix, either full or abbreviated hashes accepted). |
| TestDetail02 | 2 | partially | New file emits `new file mode`, `--- /dev/null`, `+++ b/new.txt`, and `diff --git a/new.txt b/new.txt` (destination under both prefixes). |
| TestDetail03 | 3 | no | SHAPE: same-hash mode-only change emits no `index `, no `@@`, no `--- `/`+++ ` lines — header stands alone. |
| TestDetail04 | 4 | partially | Binary patch emits no `--- `/`+++ ` pair, does emit `Binary files … differ` and still emits `index `. |
| TestDetail05 | 5 | partially | A hunk exists for a 1-line change and `,1 ` does not appear in the header — count omitted exactly at 1. |
| TestDetail06 | 6 | no | SHAPE: the hunk header line's tail after the second `@@` is empty or ` <heading>` with no embedded newline — heading may follow, never unbounded. |
| TestDetail07 | 7 | partially | A 4-line unchanged gap (ctx=3) keeps one hunk (`@@` count = 2); a 10-line gap splits into two hunks (`@@` count = 4). |
| TestDetail08 | 8 | no | SHAPE: with ctx=0, adjacent delete+add share output with no ` ` context lines emitted. |
| TestDetail09 | 9 | partially | A no-trailing-newline content line is followed by `\ No newline at end of file`. |
| TestDetail10 | 10 | partially | Output starts `the message\n` when the patch message lacks a newline — prepended before all file patches. |
| TestDetail11 | 11 | doc | `"l1\nl2\n"` delete content yields exactly 2 `-` lines (file `---` header excluded from the count) — no phantom third line. |
| TestDetail12 | 12 | partially | With colors set, the output contains the New/Old/Frag escapes and `+n` is wrapped directly in the New color — spans wrap content, reset implied by substring containment. |

Refusals/softening: line 1 does not pin the octal-mode text of `old/new mode` or rename
lines (only the index-with-mode-suffix rule); line 6 asserts only that the post-`@@` tail
is a single space-led fragment; line 12 asserts presence of the three keyed escapes and one
anchored wrap, not every span boundary.
