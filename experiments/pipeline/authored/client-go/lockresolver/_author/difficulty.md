# Why hard — lockresolver

the deepest multi-state protocol here: ttl semantics, commit vs rollback decision, async-commit secondary fan-out across regions with minCommitTS math, pessimistic persistence checks, status caching, and a resolving-set concurrency protocol. ~600 lines over two files; failure modes are silent (wrong commit ts = data corruption).
