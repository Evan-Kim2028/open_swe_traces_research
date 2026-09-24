# Bug report — brnotprot

Branch-protection lookups on unprotected branches stopped returning the
package's "branch not protected" sentinel — callers now see the raw API error
instead.

Expected: when GitHub answers a branch-protection request with its
not-protected error, the sentinel error is returned so `errors.Is` checks
work.

Got: the raw API error propagates; sentinel checks never match.

Reproduce with:

```
go test -count=1 ./github/
```

Do not use web search or any tool that accesses the internet; work only from the repository.
