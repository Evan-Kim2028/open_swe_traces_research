# Bug report — hostrunvalid

Hosted-runner creation requests missing required fields are no longer
rejected up front: incomplete requests sail through to the API and come back
as server errors instead of immediate validation failures.

Expected: requests missing any required field fail fast with a validation
error; complete requests still go through.

Got: incomplete requests are sent to the server as if they were valid.

Reproduce with:

```
go test -count=1 ./github/
```

Do not use web search or any tool that accesses the internet; work only from the repository.
