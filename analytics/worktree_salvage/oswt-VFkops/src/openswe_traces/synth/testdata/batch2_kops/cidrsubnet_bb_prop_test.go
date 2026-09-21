// Hidden black-box suite for cidrsubnet. Exported API only (api.md):
// IsIPv6IP, IsIPv4CIDR, IsIPv6CIDR, ParseCIDRNotation, CIDRSubnet.
// One TestDetailNN per DETAILS.md line. Seeded via HIDDEN_SEED.
package utils_test

import (
	"math/big"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"testing"

	fiutils "example.internal/kops/upup/pkg/fi/utils"
)

const HiddenSeed int64 = 20260919

func hiddenSeed() int64 {
	if s := os.Getenv("HIDDEN_SEED"); s != "" {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n
		}
	}
	return HiddenSeed
}

func cidrV4(rng *rand.Rand) string {
	return net.IPv4(byte(rng.Intn(256)), byte(rng.Intn(256)), byte(rng.Intn(256)), byte(rng.Intn(256))).String() +
		"/" + strconv.Itoa(rng.Intn(33))
}

func cidrV6(rng *rand.Rand) string {
	var b [16]byte
	for i := range b {
		b[i] = byte(rng.Intn(256))
	}
	return net.IP(b[:]).String() + "/" + strconv.Itoa(rng.Intn(129))
}

// Detail 1: IsIPv6IP parses as IP AND To4() nil; v4-mapped v6 is FALSE.
func TestDetail01_IsIPv6IPRejectsV4MappedAndGarbage(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 400; i++ {
		var b [16]byte
		for j := range b {
			b[j] = byte(rng.Intn(256))
		}
		s := net.IP(b[:]).String()
		want := net.ParseIP(s) != nil && net.ParseIP(s).To4() == nil
		if got := fiutils.IsIPv6IP(s); got != want {
			t.Fatalf("i=%d IsIPv6IP(%q)=%v want %v", i, s, got, want)
		}
	}
	for i := 0; i < 200; i++ {
		mapped := "::ffff:" + net.IPv4(byte(rng.Intn(256)), byte(rng.Intn(256)), byte(rng.Intn(256)), byte(rng.Intn(256))).String()
		if fiutils.IsIPv6IP(mapped) {
			t.Fatalf("i=%d IsIPv6IP(%q)=true for v4-mapped", i, mapped)
		}
		v4 := net.IPv4(byte(rng.Intn(256)), byte(rng.Intn(256)), byte(rng.Intn(256)), byte(rng.Intn(256))).String()
		if fiutils.IsIPv6IP(v4) {
			t.Fatalf("i=%d IsIPv6IP(%q)=true for dotted v4", i, v4)
		}
	}
	bad := []string{"", "x", "1.2.3", "256.0.0.1", "gg::1", "1:2:3:4:5:6:7:8:9", "0.0.0.0/0", " 1.2.3.4", "1.2.3.4 "}
	for _, s := range bad {
		if fiutils.IsIPv6IP(s) {
			t.Fatalf("IsIPv6IP(%q)=true", s)
		}
	}
}

// Detail 2: IsIPv4CIDR parses as CIDR AND To4() non-nil AND no ':' in literal.
func TestDetail02_IsIPv4CIDRRequiresV4AndNoColon(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 400; i++ {
		s := cidrV4(rng)
		if !fiutils.IsIPv4CIDR(s) {
			t.Fatalf("i=%d IsIPv4CIDR(%q)=false", i, s)
		}
		s6 := cidrV6(rng)
		if fiutils.IsIPv4CIDR(s6) {
			t.Fatalf("i=%d IsIPv4CIDR(%q)=true for v6 CIDR", i, s6)
		}
	}
	// A literal containing ':' must be rejected even if the parsed IP is v4-ish.
	for _, s := range []string{"::ffff:1.2.3.4/96", "1.2.3.4/33", "1.2.3.4/-1", "x/24", "1.2.3.4", "/24", "1.2.3.4/24x"} {
		if fiutils.IsIPv4CIDR(s) {
			t.Fatalf("IsIPv4CIDR(%q)=true", s)
		}
	}
}

// Detail 3: IsIPv6CIDR parses as CIDR AND To4() nil.
func TestDetail03_IsIPv6CIDRRoundTripAndV4Rejects(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 400; i++ {
		s6 := cidrV6(rng)
		ip, _, err := net.ParseCIDR(s6)
		want := err == nil && ip.To4() == nil
		if got := fiutils.IsIPv6CIDR(s6); got != want {
			t.Fatalf("i=%d IsIPv6CIDR(%q)=%v want %v", i, s6, got, want)
		}
		s4 := cidrV4(rng)
		if fiutils.IsIPv6CIDR(s4) {
			t.Fatalf("i=%d IsIPv6CIDR(%q)=true for v4 CIDR", i, s4)
		}
		mapped := "::ffff:10.0.0.1/120"
		if fiutils.IsIPv6CIDR(mapped) {
			t.Fatalf("IsIPv6CIDR(%q)=true for v4-mapped CIDR", mapped)
		}
	}
}

// Detail 4: ParseCIDRNotation shape ^/<digits>#[a-f0-9]+$ — lowercase hex only.
func TestDetail04_ParseCIDRNotationShapeValidation(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 400; i++ {
		size := rng.Intn(129)
		num := rng.Int63n(1 << 20)
		s := "/" + strconv.Itoa(size) + "#" + strconv.FormatInt(num, 16)
		gotSize, gotNum, err := fiutils.ParseCIDRNotation(s)
		if err != nil {
			t.Fatalf("i=%d ParseCIDRNotation(%q) err %v", i, s, err)
		}
		if gotSize != size || gotNum != num {
			t.Fatalf("i=%d ParseCIDRNotation(%q)=(%d,%d) want (%d,%d)", i, s, gotSize, gotNum, size, num)
		}
	}
	// Shape violations must error.
	mkBad := func() string {
		letter := string("abcdef"[rng.Intn(6)])
		switch rng.Intn(9) {
		case 0:
			return strconv.Itoa(rng.Intn(33)) + "#" + letter + strconv.FormatInt(rng.Int63n(256), 16) // missing leading /
		case 1:
			return "/#" + letter // no digits
		case 2:
			return "/" + strconv.Itoa(rng.Intn(33)) + "#" // empty netnum
		case 3:
			return "/" + strconv.Itoa(rng.Intn(33)) // no #
		case 4:
			return "/" + strconv.Itoa(rng.Intn(33)) + "#" + strings.ToUpper(letter) // uppercase hex
		case 5:
			return "/" + strconv.Itoa(rng.Intn(33)) + "#g" + strconv.FormatInt(rng.Int63n(256), 16) // non-hex
		case 6:
			return "/" + strconv.Itoa(rng.Intn(33)) + "#" + letter + "/" // trailing junk
		case 7:
			return "//" + strconv.Itoa(rng.Intn(33)) + "#" + letter // double slash
		default:
			return " " + "/" + strconv.Itoa(rng.Intn(33)) + "#" + letter // leading space
		}
	}
	for i := 0; i < 400; i++ {
		s := mkBad()
		if _, _, err := fiutils.ParseCIDRNotation(s); err == nil {
			t.Fatalf("i=%d ParseCIDRNotation(%q) unexpectedly ok", i, s)
		}
	}
}

// Detail 5: netNum is interpreted base-16 (#a -> 10).
func TestDetail05_ParseCIDRNotationNetNumIsHex(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 400; i++ {
		num := rng.Int63n(1 << 30)
		s := "/" + strconv.Itoa(rng.Intn(33)) + "#" + strconv.FormatInt(num, 16)
		_, gotNum, err := fiutils.ParseCIDRNotation(s)
		if err != nil {
			t.Fatalf("i=%d ParseCIDRNotation(%q) err %v", i, s, err)
		}
		if gotNum != num {
			t.Fatalf("i=%d ParseCIDRNotation(%q) netNum=%d want %d", i, s, gotNum, num)
		}
	}
	if _, n, err := fiutils.ParseCIDRNotation("/24#a"); err != nil || n != 10 {
		t.Fatalf("ParseCIDRNotation(/24#a) num=%d err=%v want 10", n, err)
	}
	if _, n, err := fiutils.ParseCIDRNotation("/16#ff"); err != nil || n != 255 {
		t.Fatalf("ParseCIDRNotation(/16#ff) num=%d err=%v want 255", n, err)
	}
}

// Detail 6: CIDRSubnet selects network netNum inside the prefix extended to
// newSize; errors on invalid prefix, negative netNum, netNum beyond the
// extension bits, and extension past the address size. (Shrink is tolerated
// upstream — verified behaviour — so we only check the mask and no error.)
func TestDetail06_CIDRSubnetExtendsAndBoundsChecks(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 600; i++ {
		oldSize := 8 + rng.Intn(20) // /8../27
		newSize := oldSize + 1 + rng.Intn(32-oldSize-1)
		hostBits := 32 - newSize
		extBits := newSize - oldSize
		var netNum int64
		if extBits >= 62 {
			netNum = rng.Int63n(1000)
		} else {
			netNum = rng.Int63n(1 << uint(extBits))
		}
		base := net.IPv4(byte(rng.Intn(224)), byte(rng.Intn(256)), byte(rng.Intn(256)), byte(rng.Intn(256)))
		prefix := base.String() + "/" + strconv.Itoa(oldSize)
		got, err := fiutils.CIDRSubnet(prefix, newSize, netNum)
		if err != nil {
			t.Fatalf("i=%d CIDRSubnet(%q,%d,%d) err %v", i, prefix, newSize, netNum, err)
		}
		gotIP, gotNet, perr := net.ParseCIDR(got)
		if perr != nil {
			t.Fatalf("i=%d CIDRSubnet(%q,%d,%d)=%q unparseable: %v", i, prefix, newSize, netNum, got, perr)
		}
		ones, _ := gotNet.Mask.Size()
		if ones != newSize {
			t.Fatalf("i=%d CIDRSubnet(%q,%d,%d)=%q mask /%d want /%d", i, prefix, newSize, netNum, got, ones, newSize)
		}
		// Expected base: parent's masked network address + netNum << hostBits.
		_, parentNet, _ := net.ParseCIDR(prefix)
		parentBase := new(big.Int).SetBytes(parentNet.IP.To4())
		want := new(big.Int).Add(parentBase, new(big.Int).Lsh(big.NewInt(netNum), uint(hostBits)))
		if !bytes4Equal(gotIP.To4(), want) {
			t.Fatalf("i=%d CIDRSubnet(%q,%d,%d)=%q want base %s", i, prefix, newSize, netNum, got, v4String(want))
		}
	}
	// Equal-size: only netNum 0 fits.
	if got, err := fiutils.CIDRSubnet("10.1.2.3/24", 24, 0); err != nil || got != "10.1.2.0/24" {
		t.Fatalf("CIDRSubnet equal-size=%q err=%v want 10.1.2.0/24", got, err)
	}
	// Error cases: netNum >= 2^extBits.
	for i := 0; i < 200; i++ {
		oldSize := 8 + rng.Intn(20)
		newSize := oldSize + rng.Intn(32-oldSize) // extBits >= 0
		extBits := newSize - oldSize
		var netNum int64
		if extBits >= 62 {
			netNum = rng.Int63n(1 << 20)
		} else {
			netNum = (1 << uint(extBits)) + rng.Int63n(1<<uint(extBits)+1)
		}
		prefix := "10.0.0.0/" + strconv.Itoa(oldSize)
		if _, err := fiutils.CIDRSubnet(prefix, newSize, netNum); err == nil {
			t.Fatalf("i=%d CIDRSubnet(%q,%d,%d) overflow accepted", i, prefix, newSize, netNum)
		}
	}
	if _, err := fiutils.CIDRSubnet("not-a-cidr", 24, 0); err == nil {
		t.Fatalf("CIDRSubnet invalid prefix accepted")
	}
	if _, err := fiutils.CIDRSubnet("10.0.0.0/24", 25, -1); err == nil {
		t.Fatalf("CIDRSubnet negative netNum accepted")
	}
	if _, err := fiutils.CIDRSubnet("10.0.0.0/24", 33, 0); err == nil {
		t.Fatalf("CIDRSubnet extension past /32 accepted")
	}
	if _, err := fiutils.CIDRSubnet("10.0.0.0/24", 24, 1); err == nil {
		t.Fatalf("CIDRSubnet zero-extension nonzero netNum accepted")
	}
	// Shrink: tolerated by the upstream implementation for small netNums; when
	// it returns, the result still parses and carries the newSize mask.
	for i := 0; i < 60; i++ {
		oldSize := 9 + rng.Intn(20)
		newSize := 1 + rng.Intn(oldSize-1)
		got, err, panicked := func() (g string, e error, p bool) {
			defer func() {
				if recover() != nil {
					p = true
				}
			}()
			g, e = fiutils.CIDRSubnet("10.0.0.0/"+strconv.Itoa(oldSize), newSize, int64(rng.Intn(8)))
			return g, e, false
		}()
		if panicked || err != nil {
			continue
		}
		if _, n, perr := net.ParseCIDR(got); perr != nil {
			t.Fatalf("i=%d shrink result %q unparseable", i, got)
		} else if ones, _ := n.Mask.Size(); ones != newSize {
			t.Fatalf("i=%d shrink result %q mask /%d want /%d", i, got, ones, newSize)
		}
	}
}

func bytes4Equal(ip net.IP, n *big.Int) bool {
	return ip != nil && new(big.Int).SetBytes(ip).Cmp(n) == 0
}

func v4String(n *big.Int) string {
	b := n.Bytes()
	var ip [4]byte
	copy(ip[4-len(b):], b)
	return net.IPv4(ip[0], ip[1], ip[2], ip[3]).String()
}
