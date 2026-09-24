# Bug report

`fi.BuildChanges` change detection is wrong: nil expected pointers are copied as changes
instead of skipped, a nil actual doesn't mark all fields changed, `CompareWithID` values
compare by pointer identity rather than ID, `Resource` fields compare by DeepEqual instead of
content (and a not-ready expected resource is treated as matching), and map/slice fields fall
back to shallow equality so nested differences are missed.

Expected: nil-in-expected means don't-care; nil actual copies everything; ID-based and
resource-content comparison; recursive map/slice equality.

Reproduce with:

```
go test -count=1 ./upup/pkg/fi/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the
internet; work only from the repository.
