# Bug report

SSH public-key fingerprints come out wrong: AWS key import rejects RSA keys or reports a fingerprint that doesn't match what AWS shows, ed25519 keys either error or get the wrong digest, and the OpenSSH fingerprint is computed over the wrong bytes or formatted without the colon separators.

Expected: an RSA public key's AWS fingerprint is a colon-separated md5 of its DER encoding; an ed25519 key's AWS fingerprint is the base64 SHA256 form; the OpenSSH fingerprint is the colon-separated md5 of the SSH wire encoding; a key string with extra whitespace or a trailing comment still parses.

Reproduce with:

```
go test -count=1 ./pkg/pki/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository.
