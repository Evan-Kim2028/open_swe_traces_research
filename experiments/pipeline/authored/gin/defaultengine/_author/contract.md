# Contract (L2) — defaultengine

A single engine is created lazily on first use — exactly once, concurrently safe — with the default middleware stack, and every package-level function delegates to that same shared instance. Route registrars (GET/POST/PUT/DELETE/PATCH/OPTIONS/HEAD/Any/Handle) return the engine's IRoutes; Group returns its RouterGroup; Use appends global middleware; NoRoute/NoMethod register fallbacks; StaticFile/Static/StaticFS register file serving; LoadHTMLGlob/LoadHTMLFiles/LoadHTMLFS/SetHTMLTemplate configure templates; Routes returns the accumulated route table; Run/RunTLS/RunUnix/RunFd forward their arguments to the engine's matching method. Because the engine is shared, state registered through any wrapper is visible through all the others (routes registered via GET appear in Routes(), middleware applies to all later requests).

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestGET..TestAny/TestHandle` | each method wrapper registers a working route on the shared engine |
| `TestGroup/TestUse` | group and middleware wrappers affect the shared engine |
| `TestNoRoute/TestNoMethod` | fallback handlers apply to unmatched requests |
| `TestRoutes` | routes registered through wrappers are listed |
| `TestSetHTMLTemplate/TestStaticFile/TestStatic/TestStaticFS` | template and static-file wrappers configure the same engine |
