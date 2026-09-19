# Contract (L2) — provenance

A signatory signs a chart archive and verifies provenance files. Keys are loaded from a secret keyring file or keybox stream (armored or binary); a passphrase callback decrypts the signing entity. ClearSign produces an OpenPGP clearsigned document whose body is a message block: `Files:` entries mapping `sha256:` sums to filenames plus the serialized chart metadata — the sum covers the archive bytes, not the metadata. Verify parses a provenance document: it extracts the clearsign block, confirms the archive's digest appears in the signed sums for the archive filename, checks the OpenPGP signature against the public keyring, and reports verification status — a digest mismatch or signature failure yields a failure status with the cause, not a panic. Message-block parsing extracts sha256 sums and metadata; malformed blocks error. Armored keyrings are decoded block by block; non-key armor blocks are rejected. Digest helpers stream a sha256 hex over files/readers.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestMessageBlock` | message block construction |
| `TestParseMessageBlock` | sums+metadata extracted |
| `TestLoadKey` | secret key load |
| `TestLoadKeyRing` | keyring load |
| `TestLoadKeyRingKeybox` | keybox ring load |
| `TestLoadKeyRingMixedKeybox` | mixed ring load |
| `TestLoadKeyRingArmored` | armored ring load |
| `TestLoadKeyRingArmoredMultiBlock` | multi-block armor |
| `TestLoadArmoredKeyRingRejectsNonKeyBlocks` | non-key blocks rejected |
| `TestDigest` | stream digest |
| `TestNewFromFiles` | signatory from key files |
| `TestDigestFile` | file digest |
| `TestDecryptKey` | passphrase decrypt |
| `TestClearSign` | clearsign output |
| `TestMixedKeyringRSASigningAndVerification` | sign+verify round trip |
| `TestClearSignError` | signing errors |
| `TestVerify` | verification status |
| `TestVerifyKeyboxKeyring` | verify via keybox |



Reproduce with:

```
tests/test.sh
```

That script installs the hidden suite and runs it (equivalent to
`go test -count=1 -timeout 15m ./pkg/provenance/`
after the suite is in the tree). Do not skip, delete, or weaken the
tests. Do not change test assertions or testdata just to make them
green.

Work in `/app`. Keep unrelated tests passing.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
