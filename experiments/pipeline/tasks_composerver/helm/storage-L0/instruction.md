# Bug report

Installing or upgrading a release fails when writing release bookkeeping: `helm install`/`upgrade`/`history`/`list`/`uninstall` report storage errors, history is empty, and pruning of old revisions never happens or deletes the wrong revision.

Reproduce with:

```
go test -count=1 ./pkg/storage/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
