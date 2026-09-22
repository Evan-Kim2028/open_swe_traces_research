# Details — dmfopts

1. `WithIntervalFeatOpt(d)` returns a non-nil `FeatOpt`; applying it to a
   `featCfg` sets the interval field to exactly `d` — no clamping, no
   default substitution. Inferable: doc — "updates the up time for the
   feature".
2. `WithSyncFSFeatOpt(b)` returns a non-nil `FeatOpt`; applying it sets the
   syncFS field to exactly `b`. Inferable: doc — "determine if the caller
   wants to synchronize filesystem".
3. The two options are independent: applying the interval option leaves
   syncFS untouched, and applying the syncFS option leaves interval
   untouched. Inferable: partially — orthogonal options are the point of
   the pattern; not stated anywhere.
4. `validateFSType` returns nil for exactly the two declared constants
   `FSTypeEXT4` and `FSTypeXFS`, and a non-nil error for any other value,
   including the empty `FSType("")`. Inferable: doc — the const block is
   marked "Supported filesystems".
5. The unsupported-fsType error's message contains the offending value.
   Inferable: no.
6. `flakey.DevicePath()` is a pure join: the literal prefix "/dev/mapper/"
   plus the device's own name, on every call, with no state lookup or I/O.
   Inferable: doc — "The device-mapper device will be
   /dev/mapper/$flakeyDevice".
7. `flakey.Filesystem()` returns the fsType stored at construction — a pure
   getter with no computation. Inferable: doc.
8. `createEmptyFSImage` validates `fsType` FIRST: on an unsupported fsType
   it returns an error before creating the image file, its parent
   directory, or invoking any external tool. Inferable: partially — that
   validation happens is inferable from validateFSType existing; the
   ordering is not stated.
9. `createEmptyFSImage` refuses to overwrite: if `imgPath` already exists
   it returns a non-nil error and does not truncate the existing file
   (reachable only once `mkfs.<fstype>` has been located). Inferable:
   partially — "creates empty filesystem" implies no-clobber intent; the
   guard itself is not documented.
