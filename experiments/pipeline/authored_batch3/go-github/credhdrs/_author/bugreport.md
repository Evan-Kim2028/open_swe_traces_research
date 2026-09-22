# Bug report — credhdrs

Authenticated transports stopped sending credentials: requests made through
the basic-auth and unauthenticated-rate-limited transports arrive at the API
with no Authorization header, so everything comes back 401.

Expected: each outgoing request carries the configured credentials in its
headers while the caller's original request object stays untouched.

Got: requests go out unauthenticated.

Reproduce with:

```
go test -count=1 ./github/
```

Do not use web search or any tool that accesses the internet; work only from the repository.
