# Missing behavior

Outgoing calls can be wrapped by a stack of decorators. Each decorator runs
its before-logic, then the next decorator, then the real call, then returns
in reverse. The stack is an onion: the decorator attached first runs first
and returns last.

Two decorators that share a name do not both stay on the stack; the later
attachment replaces the earlier one. Composing two distinct decorators is
the same as attaching the first argument, then the second.

A helper can bind a decorator onto a context value and later retrieve it.
A context that was never bound yields no decorator.

Worked cases the hidden tests assert:

- Attach `first` then `second`, then invoke: the trace is first, second, base.
- Attach the same name twice: the stack length is 1, not 2.
- Bind a decorator on a context, read it back; an empty context reads as none.
- expected order first then second then base, actual only the base call (a
  no-op wrapper that never invokes the attached decorators).

Coverage the hidden checks enforce:

- Decorators run in attachment order, then the base call; N attachments yield N+1 trace entries.
- Same-name attachments keep a single entry; composing two distinct decorators runs the first argument then the second.
- Context bind/retrieve round-trips; an unbound context yields none.
- Random unseen names still run in argument order; hardcoding a first/second pair fails.

Reproduce with:

```
go test -count=1 -timeout 15m ./wirerpc/interceptor/
```

Implement the missing behavior so the hidden tests pass. Do not skip,
delete, or weaken the tests. Do not change test assertions or testdata
just to make them green.

Work in `/app`. Keep unrelated tests passing. The fix must not break
existing behavior.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
