# Bug report — signmsg

Signed commits can't be produced: asking for a signature always errors out
even with a valid signer and a complete commit, so callers that require
verified commits are blocked.

Expected: a commit with the required fields is serialized and handed to the
signer, producing a real signature; incomplete inputs still fail cleanly.

Got: signing fails unconditionally.

Reproduce with:

```
go test -count=1 ./github/
```

Do not use web search or any tool that accesses the internet; work only from the repository.
