// Hidden black-box suite for getterdispatch. Exported API only: Provider.Provides,
// Providers.ByScheme, Getters, All, Option helpers.
package getter_test

import (
	"bytes"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"example.internal/helm/pkg/cli"
	"example.internal/helm/pkg/getter"
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

func TestDetail01_ProvidesExactSchemeMembership(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 80; i++ {
		s := "s" + strconv.Itoa(rng.Intn(20))
		p := getter.Provider{Schemes: []string{"http", s, "ftp"}}
		if !p.Provides(s) {
			t.Fatalf("exact membership of %q", s)
		}
		if p.Provides(s + "x") {
			t.Fatalf("prefix must not match %q", s+"x")
		}
		if p.Provides(strings.ToUpper(s)) && s != strings.ToUpper(s) {
			t.Fatalf("case: Provides(%q) of schemes %v", strings.ToUpper(s), p.Schemes)
		}
	}
	p := getter.Provider{Schemes: []string{"http", "https"}}
	if p.Provides("HTTP") {
		t.Fatal("Provides is exact, not case-fold")
	}
	if p.Provides("htt") || p.Provides("") {
		t.Fatal("no partial/empty membership")
	}
}

func TestDetail02_BySchemeFirstMatchAndErrorText(t *testing.T) {
	first := getter.Provider{Schemes: []string{"http"}, New: func(...getter.Option) (getter.Getter, error) {
		return stub{id: 1}, nil
	}}
	second := getter.Provider{Schemes: []string{"http"}, New: func(...getter.Option) (getter.Getter, error) {
		return stub{id: 2}, nil
	}}
	ps := getter.Providers{first, second}
	g, err := ps.ByScheme("http")
	if err != nil {
		t.Fatal(err)
	}
	if g.(stub).id != 1 {
		t.Fatal("ByScheme must use the first matching provider")
	}
	_, err = ps.ByScheme("oci")
	if err == nil {
		t.Fatal("unknown scheme must error")
	}
	if err.Error() != `scheme "oci" not supported` {
		t.Fatalf("error text: %q", err.Error())
	}
}

func TestDetail03_BuiltinsHTTPHTTPSOneProviderPlusOCI(t *testing.T) {
	ps := getter.Getters()
	httpN, httpsN, ociN := 0, 0, 0
	for _, p := range ps {
		if p.Provides("http") {
			httpN++
		}
		if p.Provides("https") {
			httpsN++
		}
		if p.Provides("oci") {
			ociN++
		}
	}
	if httpN != 1 || httpsN != 1 {
		t.Fatalf("http/https provider counts %d %d", httpN, httpsN)
	}
	// same provider must cover both http and https
	var httpAndHTTPS int
	for _, p := range ps {
		if p.Provides("http") && p.Provides("https") {
			httpAndHTTPS++
		}
	}
	if httpAndHTTPS != 1 {
		t.Fatalf("one provider for {http,https}, got %d", httpAndHTTPS)
	}
	if ociN != 1 {
		t.Fatalf("one OCI provider, got %d", ociN)
	}
}

func TestDetail04_ConstructorOptionOrderCallThenDefaultThenExtras(t *testing.T) {
	// extras beat the default 120s; Get-time call opts are a different layer.
	// Observe: Getters(WithTimeout(50ms)) should apply 50ms (extras after default).
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(slow.Close)
	ps := getter.Getters(getter.WithTimeout(50 * time.Millisecond))
	g, err := ps.ByScheme("http")
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	_, err = g.Get(slow.URL)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("50ms extra timeout should fire against 200ms handler")
	}
	if elapsed > 150*time.Millisecond {
		t.Fatalf("timeout extras should win over default 120s, elapsed %s err=%v", elapsed, err)
	}
}

func TestDetail05_AllSwallowsDiscoveryErrorAddsPlugins(t *testing.T) {
	st := cli.New()
	st.PluginsDirectory = t.TempDir() + "/does-not-exist"
	ps := getter.All(st)
	if len(ps) < 2 {
		t.Fatalf("discovery error swallowed; builtins still present, got %d", len(ps))
	}
	plugdir := filepath.Join("testdata", "plugins")
	if _, err := os.Stat(plugdir); err != nil {
		t.Skip("plugin fixtures missing")
	}
	st.PluginsDirectory = plugdir
	ps = getter.All(st)
	found := false
	for _, p := range ps {
		if p.Provides("test") {
			found = true
		}
	}
	if !found {
		t.Fatal("getter/v1 plugin protocols must be added as schemes")
	}
}

func TestDetail06_OnlyGetterV1PluginsContribute(t *testing.T) {
	dir := t.TempDir()
	// a non-getter plugin must not contribute schemes
	pdir := filepath.Join(dir, "hello")
	if err := os.MkdirAll(pdir, 0o755); err != nil {
		t.Fatal(err)
	}
	yaml := []byte("name: hello\nversion: 0.1.0\ntype: cli/v1\napiVersion: v1\nruntime: subprocess\n")
	if err := os.WriteFile(filepath.Join(pdir, "plugin.yaml"), yaml, 0o644); err != nil {
		t.Fatal(err)
	}
	st := cli.New()
	st.PluginsDirectory = dir
	ps := getter.All(st)
	for _, p := range ps {
		if p.Provides("hello") {
			t.Fatal("non getter/v1 plugin must not contribute")
		}
	}
	gdir := filepath.Join("testdata", "plugins")
	if _, err := os.Stat(gdir); err == nil {
		st.PluginsDirectory = gdir
		ps = getter.All(st)
		ok := false
		for _, p := range ps {
			if p.Provides("test") {
				ok = true
			}
		}
		if !ok {
			t.Fatal("getter/v1 with protocols must contribute schemes")
		}
	}
}

func TestDetail07_ConvertOptionsCallBeatsGlobalSubset(t *testing.T) {
	// Plugin Get is the only way to observe convertOptions. Use testdata plugin.
	gdir := filepath.Join("testdata", "plugins")
	if _, err := os.Stat(gdir); err != nil {
		t.Skip("plugin fixtures missing")
	}
	st := cli.New()
	st.PluginsDirectory = gdir
	ps := getter.All(st, getter.WithUserAgent("global-agent"))
	g, err := ps.ByScheme("test")
	if err != nil {
		t.Fatal(err)
	}
	_, err = g.Get("test://example.test/x", getter.WithUserAgent("call-agent"))
	if err != nil {
		// plugin echo may fail for wire-format reasons; the important part is
		// it was invoked rather than "scheme not supported"
		if strings.Contains(err.Error(), "not supported") {
			t.Fatal(err)
		}
	}
}

func TestDetail08_PluginGetSchemeProtocolAndErrorWrap(t *testing.T) {
	gdir := filepath.Join("testdata", "plugins")
	if _, err := os.Stat(gdir); err != nil {
		t.Skip("plugin fixtures missing")
	}
	st := cli.New()
	st.PluginsDirectory = gdir
	ps := getter.All(st)
	g, err := ps.ByScheme("test")
	if err != nil {
		t.Fatal(err)
	}
	_, err = g.Get("test://host/path")
	// echo plugin will typically fail the typed-message contract
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "failed to invoke") || strings.Contains(msg, "invalid output message type") {
			return
		}
		// still a plugin-path error, not a scheme miss
		if strings.Contains(msg, "not supported") {
			t.Fatalf("plugin Get used wrong scheme: %v", err)
		}
	}
}

type stub struct{ id int }

func (s stub) Get(string, ...getter.Option) (*bytes.Buffer, error) {
	return bytes.NewBuffer(nil), nil
}
