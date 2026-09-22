The on-disk account JWT resolver stopped finding files it just wrote.

Saving a JWT for a normal public nkey used to create `<store-dir>/<pubkey>.jwt` and a later fetch of that same key would read it back. Keys with fewer than two characters were rejected (load failed as invalid). After the last change every save/load reports an invalid public key, including keys that `nkeys` still accepts, so resolver updates never land on disk.
