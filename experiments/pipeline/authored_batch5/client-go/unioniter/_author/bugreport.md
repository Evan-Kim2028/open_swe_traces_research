# Bug report

`internal/unionstore`'s union iterator panics: `NewUnionIter`, the merge
step (`updateCur`/`Next`), `Key`/`Value`/`Valid` and `Close` are stubbed, so
`BufferStore.Iter`/`IterReverse` panic for every scan.

Expected (in-package probes with fake child iterators): dirty {b=B, f=F}
over snapshot {a=A, d=D, e=E} yields `[a=A b=B e=E f=F]`; equal keys take
the dirty value ({m=M2} over {m=M1} yields `[m=M2]`); in reverse mode with
descending children {d,b} over {d,a} the merge yields `[b=B a=A]`. An
empty-value dirty entry deletes the snapshot record — it never appears.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
