# Bug report

gRPC handlers panic when a typed application error is returned, or they send the wrong status (a missing field looks like an internal fault; a timeout is not a deadline). Merged validation errors lose the individual field names. A canceled caller still looks like a generic transport failure, or a join of two independent failures is collapsed into one cancellation. Clients cannot recover the original detail message.

Reproduce with:

```
go test -count=1 ./grpc/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository and test output.
