# Exported API — dmfopts

Package `dmflakey` (module `example.internal/boltstore`, `tests/dmflakey/`) —
the configuration and description surface of the device-mapper fault-injection
helper, plus its filesystem-image validation path.

`WithIntervalFeatOpt(interval time.Duration) FeatOpt`,
`WithSyncFSFeatOpt(syncFS bool) FeatOpt` — option constructors for the
`AllowWrites`/`DropWrites`/`ErrorWrites` feature calls.

`validateFSType(fsType FSType) error` — gate on the supported-filesystem set.

`(*flakey).DevicePath() string` — the `/dev/mapper/...` path of the device.
`(*flakey).Filesystem() FSType` — the filesystem the device was built with.

`createEmptyFSImage(imgPath string, fsType FSType, mkfsOpt string) error` —
validates the fs type, locates `mkfs.<fstype>`, refuses to overwrite an
existing image, then creates and formats a sparse image file.

Callers: `InitFlakey` (same file, kept) and the dmflakey/robustness test
drivers. In-tree tests removed: 0 (existing tests are root-gated and pass
vacuously; hidden tests exercise this surface in-package).
