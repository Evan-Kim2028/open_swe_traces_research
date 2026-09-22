1. A public key shorter than 2 bytes yields an empty path. Inferable: yes
2. A string that fails `nkeys.IsValidPublicKey` yields an empty path. Inferable: yes
3. The file base name is the public key concatenated with `fileExtension` (`.jwt`). Inferable: yes
4. When `store.shard` is false the path is `filepath.Join(store.directory, fileName)`. Inferable: yes
5. When `store.shard` is true the path is `filepath.Join(store.directory, lastTwo, fileName)` where `lastTwo` is the final two characters of the public key. Inferable: no
6. Sharding uses the suffix of the key, not a hash and not the prefix. Inferable: no
7. Empty is the only failure result; the function does not return an error. Inferable: yes
