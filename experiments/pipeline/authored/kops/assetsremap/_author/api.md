# Exported API — assetsremap

`NewAssetBuilder(vfsContext, assets, getAssets) *AssetBuilder`

`(*AssetBuilder) RemapImage(image string) string` — rewrite a container image for proxy/registry and record it.

`(*AssetBuilder) RemapManifest(data []byte) ([]byte, error)` — YAML objects, remap every image, emit YAML.

`(*AssetBuilder) RemapFile(canonicalURL *url.URL, knownHash *hashing.Hash) (*FileAsset, error)` — optional file-repository rewrite plus SHA.

`NormalizeImage(a *AssetBuilder, image string) string`

`(*AssetBuilder) ImageAssets() []*ImageAsset` / `FileAssets() []*FileAsset` — sorted snapshots.

Callers: channel/bootstrap builders and nodeup asset staging. In-tree tests call RemapImage/RemapFile/RemapManifest and the snapshot getters.
