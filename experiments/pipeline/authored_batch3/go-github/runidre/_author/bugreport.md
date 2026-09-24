# Bug report — runidre

Extracting the workflow run ID from a deployment-protection event always
fails: valid callback URLs report "no match" instead of the run number, so
protection-rule handlers cannot correlate callbacks to runs.

Expected: callback URLs in the documented deployment-protection shape yield
the numeric run ID; anything else still reports no match.

Got: every URL reports no match.

Reproduce with:

```
go test -count=1 ./github/
```

Do not use web search or any tool that accesses the internet; work only from the repository.
