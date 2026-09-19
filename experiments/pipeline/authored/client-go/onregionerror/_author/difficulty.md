# Why hard — onregionerror

largest sequence contract in the repo: ~10 region-error classes each with a different retry/backoff/invalidation reaction, a send-failure path distinct from region errors, token-limited retries, and interaction with the replica selector and cache invalidation. A solver must rediscover the full decision table from nothing but observable behavior.
