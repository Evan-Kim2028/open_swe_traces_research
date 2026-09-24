package server

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/nats-io/nkeys"
)

func jpkKey(t *testing.T) string {
	t.Helper()
	kp, err := nkeys.CreateAccount()
	if err != nil {
		t.Fatalf("nkeys.CreateAccount: %v", err)
	}
	pub, err := kp.PublicKey()
	if err != nil {
		t.Fatalf("PublicKey: %v", err)
	}
	return pub
}

// TestDetail01 (yes): a public key shorter than 2 bytes yields an empty path.
func TestDetail01(t *testing.T) {
	store := &DirJWTStore{directory: t.TempDir()}
	for _, k := range []string{"", "A"} {
		if got := store.pathForKey(k); got != _EMPTY_ {
			t.Fatalf("pathForKey(%q) = %q, want empty", k, got)
		}
	}
}

// TestDetail02 (yes): a string failing nkeys.IsValidPublicKey yields an empty
// path.
func TestDetail02(t *testing.T) {
	store := &DirJWTStore{directory: t.TempDir()}
	// A real key with the last character swapped fails the checksum.
	pub := jpkKey(t)
	bad := pub[:len(pub)-1]
	if pub[len(pub)-1] == 'A' {
		bad += "B"
	} else {
		bad += "A"
	}
	for _, k := range []string{"notakey", bad} {
		if nkeys.IsValidPublicKey(k) {
			t.Fatalf("test premise broken: %q is a valid public key", k)
		}
		if got := store.pathForKey(k); got != _EMPTY_ {
			t.Fatalf("pathForKey(%q) = %q, want empty", k, got)
		}
	}
}

// TestDetail03 (yes): the file base name is the public key + ".jwt".
func TestDetail03(t *testing.T) {
	store := &DirJWTStore{directory: t.TempDir()}
	pub := jpkKey(t)
	got := store.pathForKey(pub)
	if filepath.Base(got) != pub+".jwt" {
		t.Fatalf("base name = %q, want %q", filepath.Base(got), pub+".jwt")
	}
}

// TestDetail04 (yes): shard=false -> filepath.Join(directory, fileName).
func TestDetail04(t *testing.T) {
	dir := t.TempDir()
	store := &DirJWTStore{directory: dir}
	pub := jpkKey(t)
	want := filepath.Join(dir, pub+".jwt")
	if got := store.pathForKey(pub); got != want {
		t.Fatalf("pathForKey = %q, want %q", got, want)
	}
}

// TestDetail05 (shape — Inferable: no): shard=true ->
// directory / <two-char tail of the key> / <key>.jwt — one extra level named
// by a short substring taken from the key's tail.
func TestDetail05(t *testing.T) {
	dir := t.TempDir()
	store := &DirJWTStore{directory: dir, shard: true}
	pub := jpkKey(t)
	got := store.pathForKey(pub)

	want := filepath.Join(dir, pub[len(pub)-2:], pub+".jwt")
	if got != want {
		t.Fatalf("sharded path = %q, want %q", got, want)
	}
	// The sharding directory is a short fragment of the key, not the whole
	// key and not a hash of it.
	mid := filepath.Base(filepath.Dir(got))
	if len(mid) != 2 || !strings.Contains(pub, mid) {
		t.Fatalf("shard dir %q is not a 2-char fragment of the key", mid)
	}
}

// TestDetail06 (shape — Inferable: no): sharding uses the suffix of the key,
// not a hash and not the prefix.
func TestDetail06(t *testing.T) {
	dir := t.TempDir()
	store := &DirJWTStore{directory: dir, shard: true}
	pub := jpkKey(t)
	if pub[:2] == pub[len(pub)-2:] {
		t.Skip("generated key has identical prefix and suffix; rerun")
	}
	got := store.pathForKey(pub)
	mid := filepath.Base(filepath.Dir(got))
	if mid == pub[:2] {
		t.Fatalf("shard dir used the key prefix %q", mid)
	}
	if mid != pub[len(pub)-2:] {
		t.Fatalf("shard dir = %q, want the key's last two characters %q", mid, pub[len(pub)-2:])
	}
}

// TestDetail07 (yes): empty is the only failure result; the function does
// not return an error.
func TestDetail07(t *testing.T) {
	store := &DirJWTStore{directory: t.TempDir()}
	if got := store.pathForKey(jpkKey(t)); got == _EMPTY_ {
		t.Fatal("valid key produced an empty path")
	}
}
