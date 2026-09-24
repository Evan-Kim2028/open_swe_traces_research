package object

import (
	_ "bytes"
	"io"

	"example.internal/gitkit/v6/plumbing"
	_ "example.internal/gitkit/v6/utils/ioutil"
	_ "example.internal/gitkit/v6/utils/sync"
)

const (
	signatureTypeUnknown signatureType = iota
	signatureTypeOpenPGP
	signatureTypeX509
	signatureTypeSSH
)

var (
	// openPGPSignatureFormat is the format of an OpenPGP signature.
	openPGPSignatureFormat = signatureFormat{
		[]byte("-----BEGIN PGP SIGNATURE-----"),
		[]byte("-----BEGIN PGP MESSAGE-----"),
	}
	// x509SignatureFormat is the format of an X509 signature, which is
	// a PKCS#7 (S/MIME) signature.
	x509SignatureFormat = signatureFormat{
		[]byte("-----BEGIN SIGNED MESSAGE-----"),
	}

	// sshSignatureFormat is the format of an SSH signature.
	sshSignatureFormat = signatureFormat{
		[]byte("-----BEGIN SSH SIGNATURE-----"),
	}
)

// knownSignatureFormats is a map of known signature formats, indexed by
// their signatureType.
var knownSignatureFormats = map[signatureType]signatureFormat{
	signatureTypeOpenPGP: openPGPSignatureFormat,
	signatureTypeX509:    x509SignatureFormat,
	signatureTypeSSH:     sshSignatureFormat,
}

// signatureType represents the type of the signature.
type signatureType int8

// signatureFormat represents the beginning of a signature.
type signatureFormat [][]byte

// typeForSignature returns the type of the signature based on its format.
func typeForSignature(b []byte) signatureType {
	panic("excised: typeForSignature")
}

// parseSignedBytes returns the position of the last signature block found in
// the given bytes. If no signature block is found, it returns -1.
//
// When multiple signature blocks are found, the position of the last one is
// returned. Any tailing bytes after this signature block start should be
// considered part of the signature.
//
// Given this, it would be safe to use the returned position to split the bytes
// into two parts: the first part containing the message, the second part
// containing the signature.
//
// Example:
//
//	message := []byte(`Message with signature
//
//	-----BEGIN SSH SIGNATURE-----
//	...`)
//
//	var signature string
//	if pos, _ := parseSignedBytes(message); pos != -1 {
//		signature = string(message[pos:])
//		message = message[:pos]
//	}
//
// This logic is on par with git's gpg-interface.c:parse_signed_buffer().
// https://github.com/git/git/blob/7c2ef319c52c4997256f5807564523dfd4acdfc7/gpg-interface.c#L668
func parseSignedBytes(b []byte) (int, signatureType) {
	panic("excised: parseSignedBytes")
}

// countSignatureBlocks reports how many distinct armored signature blocks
// start at a line boundary in b. Used by verification paths to reject
// multi-signature payloads, matching upstream's check in gpg-interface.c
// where parse_gpg_output bails out the first time it sees a second
// exclusive status line (a second GOODSIG/BADSIG/etc.).
func countSignatureBlocks(b []byte) int {
	panic("excised: countSignatureBlocks")
}

// isSignatureHeader reports whether line is a canonical "gpgsig "/
// "gpgsig-sha256 " header line. Other "gpgsig"-prefixed extra headers
// are intentionally not matched.
func isSignatureHeader(line []byte) bool {
	panic("excised: isSignatureHeader")
}

// stripObjectSignatures streams src into dst, producing the byte sequence
// over which a PGP/GPG signature is computed:
//
//   - Canonical "gpgsig" and "gpgsig-sha256" headers (and their
//     continuation lines) are dropped, mirroring upstream's
//     remove_signature in commit.c.
//   - For tag objects, the inline trailing PGP signature is additionally
//     truncated, mirroring upstream's parse_signature in gpg-interface.c
//     used by gpg_verify_tag.
//
// The returned object's type is set to objType. Used by both
// Commit.EncodeWithoutSignature and Tag.EncodeWithoutSignature to
// reproduce the exact bytes the signature was computed over.
func stripObjectSignatures(dst, src plumbing.EncodedObject, objType plumbing.ObjectType) (err error) {
	panic("excised: stripObjectSignatures")
}

// stripHeaderSignatures copies r to w, dropping canonical signature header
// lines (gpgsig and gpgsig-sha256) and their continuation lines. Lines
// past the blank line that closes the header block are copied verbatim.
func stripHeaderSignatures(w io.Writer, r io.Reader) error {
	panic("excised: stripHeaderSignatures")
}
