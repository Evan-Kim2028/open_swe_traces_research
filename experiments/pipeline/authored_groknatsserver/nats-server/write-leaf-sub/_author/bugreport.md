Leafnode interest updates stopped propagating.

A hub used to send `LS+ foo` when a leaf gained a plain subscription on `foo`, `LS+ foo bar 3` when a queue subscription `foo` / `bar` had three members, and `LS- foo` when interest dropped to zero. Empty keys used to be skipped.

After the last change the hub writes nothing on those updates, so the spoke never sees remote interest and leaf traffic is dropped as unsubscribed.
