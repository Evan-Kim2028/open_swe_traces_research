# Why hard — memdbstaging

checkpointed write-ahead over an arena-allocated red-black tree: levels must merge/discard in the right order, flags and tombstones must survive correctly, value-log offsets must be exact for revert. Easy to write something that passes get/set and corrupts nested staging.
