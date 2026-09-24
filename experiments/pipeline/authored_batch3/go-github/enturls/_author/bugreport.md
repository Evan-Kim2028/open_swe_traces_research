# Bug report — enturls

Enterprise clients are hitting the wrong endpoints: after configuring the
client for a GitHub Enterprise host, API calls go to the bare host instead of
the instance's API path, and uploads miss their upload path entirely.
Non-enterprise custom URLs also lose their trailing slash.

Expected: enterprise configuration yields the conventional API and upload
paths (with exemptions for already-conventional hosts), and every configured
URL ends in a trailing slash; empty or unparseable URLs are rejected.

Got: URLs are used verbatim — no API path, no upload path, no trailing-slash
normalization, and empty URLs slip through.

Reproduce with:

```
go test -count=1 ./github/
```

Do not use web search or any tool that accesses the internet; work only from the repository.
