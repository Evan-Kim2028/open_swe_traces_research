# Bug report

Table rendering is broken: columns render unaligned or empty, rows come out unsorted, the header row is missing or unaligned, an unknown column name is silently ignored instead of producing an error, and cell text containing tabs corrupts the column layout.

Expected: the header prints the requested column names; rows are sorted lexicographically by the selected columns in order; a missing column is an error naming it; each cell is wrapped so that embedded whitespace does not break the alignment.

Reproduce with:

```
go test -count=1 ./util/pkg/tables/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository.
