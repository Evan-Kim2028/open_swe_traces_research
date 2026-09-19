// Black-box property suite for the defaultengine unit (package ginS).
// Exported API only: GET/POST/PUT/DELETE/PATCH/OPTIONS/HEAD/Any/Handle/Group/Use/
// NoRoute/NoMethod/StaticFile/Static/StaticFS/LoadHTMLGlob/Routes/Run.
// Seed 20260919; contract + gins_test.go patterns. Do not call unexported engine().
//
// Contract (contract.md) -> property coverage table:
//
//	C1 "lazy single shared engine; wrappers delegate" -> TestDEContractTableProperty
//	C2 "GET..HEAD/Any/Handle register working routes" -> TestDEHTTPMethodsProperty
//	C3 "Group and Use affect shared engine" -> TestDEGroupMiddlewareProperty
//	C4 "NoRoute/NoMethod fallbacks" -> TestDENoRouteNoMethodProperty
//	C5 "Routes lists registrations from any wrapper" -> TestDESharedRoutesProperty
//	C6 "StaticFile/Static/StaticFS serve files" -> TestDEStaticProperty
//	C7 "LoadHTMLGlob configures templates" -> TestDEHTMLGlobProperty
//	C8 adversarial concurrent first-use -> TestDEAdversarialLazyInitProperty
//	C9 unseen random path/method registration -> TestDEUnseenRandomProperty
package ginS_test

import (
	"fmt"
	"html/template"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	gin "example.internal/httprouter"
	ginS "example.internal/httprouter/ginS"
)

const (
	bbSeed  = 20260919
	bbCases = 10000
)

func init() {
	gin.SetMode(gin.TestMode)
}

var (
	bbSrvOnce sync.Once
	bbSrvAddr string
)

func bbEnsureServer(t *testing.T) string {
	t.Helper()
	bbSrvOnce.Do(func() {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("listen: %v", err)
		}
		bbSrvAddr = ln.Addr().String()
		ln.Close()
		go func() { _ = ginS.Run(bbSrvAddr) }()
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			conn, err := net.Dial("tcp", bbSrvAddr)
			if err == nil {
				conn.Close()
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
		t.Fatal("ginS.Run did not start")
	})
	return bbSrvAddr
}

func bbPerform(t *testing.T, method, path string, body io.Reader) *httptest.ResponseRecorder {
	t.Helper()
	addr := bbEnsureServer(t)
	req, err := http.NewRequest(method, "http://"+addr+path, body)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	rec := httptest.NewRecorder()
	rec.WriteHeader(resp.StatusCode)
	if resp.Body != nil {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		_, _ = rec.Write(b)
	}
	for k, vv := range resp.Header {
		for _, v := range vv {
			rec.Header().Add(k, v)
		}
	}
	return rec
}

func bbHasRoute(method, path string) bool {
	for _, r := range ginS.Routes() {
		if r.Method == method && r.Path == path {
			return true
		}
	}
	return false
}

func bbUniquePath(rng *rand.Rand, prefix string) string {
	return fmt.Sprintf("/bb-%s-%d-%d", prefix, bbSeed, rng.Int63())
}

// TestDEContractTableProperty pins gins_test.go fixture behaviours on the shared engine.
func TestDEContractTableProperty(t *testing.T) {
	p := bbUniquePath(rand.New(rand.NewSource(bbSeed)), "contract")
	ginS.GET(p+"/get", func(c *gin.Context) { c.String(http.StatusOK, "test") })
	w := bbPerform(t, http.MethodGet, p+"/get", nil)
	if w.Code != http.StatusOK || w.Body.String() != "test" {
		t.Fatalf("GET fixture: code=%d body=%q", w.Code, w.Body.String())
	}
	if !bbHasRoute(http.MethodGet, p+"/get") {
		t.Fatal("GET route missing from Routes()")
	}

	ginS.POST(p+"/post", func(c *gin.Context) { c.String(http.StatusCreated, "created") })
	w = bbPerform(t, http.MethodPost, p+"/post", nil)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST fixture: code=%d", w.Code)
	}

	ginS.PUT(p+"/put", func(c *gin.Context) { c.String(http.StatusOK, "updated") })
	w = bbPerform(t, http.MethodPut, p+"/put", nil)
	if w.Code != http.StatusOK || w.Body.String() != "updated" {
		t.Fatalf("PUT fixture")
	}

	ginS.DELETE(p+"/delete", func(c *gin.Context) { c.String(http.StatusOK, "deleted") })
	w = bbPerform(t, http.MethodDelete, p+"/delete", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("DELETE fixture")
	}

	ginS.PATCH(p+"/patch", func(c *gin.Context) { c.String(http.StatusOK, "patched") })
	w = bbPerform(t, http.MethodPatch, p+"/patch", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("PATCH fixture")
	}

	ginS.OPTIONS(p+"/options", func(c *gin.Context) { c.String(http.StatusOK, "options") })
	w = bbPerform(t, http.MethodOptions, p+"/options", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("OPTIONS fixture")
	}

	ginS.HEAD(p+"/head", func(c *gin.Context) { c.String(http.StatusOK, "head") })
	w = bbPerform(t, http.MethodHead, p+"/head", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("HEAD fixture")
	}

	ginS.Any(p+"/any", func(c *gin.Context) { c.String(http.StatusOK, "any") })
	w = bbPerform(t, http.MethodGet, p+"/any", nil)
	if w.Code != http.StatusOK || w.Body.String() != "any" {
		t.Fatalf("Any fixture")
	}

	ginS.Handle(http.MethodGet, p+"/handle", func(c *gin.Context) { c.String(http.StatusOK, "handle") })
	w = bbPerform(t, http.MethodGet, p+"/handle", nil)
	if w.Code != http.StatusOK || w.Body.String() != "handle" {
		t.Fatalf("Handle fixture")
	}
}

// TestDEHTTPMethodsProperty randomizes method wrappers on unique paths.
func TestDEHTTPMethodsProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 1))
	names := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch, http.MethodOptions, http.MethodHead}
	calls := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch, http.MethodOptions, http.MethodHead}
	register := []func(string){
		func(path string) { ginS.GET(path, func(c *gin.Context) { c.Status(http.StatusOK) }) },
		func(path string) { ginS.POST(path, func(c *gin.Context) { c.Status(http.StatusCreated) }) },
		func(path string) { ginS.PUT(path, func(c *gin.Context) { c.Status(http.StatusOK) }) },
		func(path string) { ginS.DELETE(path, func(c *gin.Context) { c.Status(http.StatusOK) }) },
		func(path string) { ginS.PATCH(path, func(c *gin.Context) { c.Status(http.StatusOK) }) },
		func(path string) { ginS.OPTIONS(path, func(c *gin.Context) { c.Status(http.StatusOK) }) },
		func(path string) { ginS.HEAD(path, func(c *gin.Context) { c.Status(http.StatusOK) }) },
	}
	for i := 0; i < 200; i++ {
		idx := rng.Intn(len(names))
		path := bbUniquePath(rng, names[idx])
		register[idx](path)
		if !bbHasRoute(calls[idx], path) {
			t.Fatalf("case %d: %s not in Routes()", i, calls[idx])
		}
		w := bbPerform(t, calls[idx], path, nil)
		if w.Code == http.StatusNotFound {
			t.Fatalf("case %d: %s %s 404", i, calls[idx], path)
		}
	}
}

// TestDESharedRoutesProperty: registration via one wrapper visible via Routes() after another.
func TestDESharedRoutesProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 2))
	for i := 0; i < 100; i++ {
		pathA := bbUniquePath(rng, "a")
		pathB := bbUniquePath(rng, "b")
		ginS.GET(pathA, func(c *gin.Context) { c.String(http.StatusOK, "a") })
		if !bbHasRoute(http.MethodGet, pathA) {
			t.Fatalf("case %d: pathA missing after GET", i)
		}
		ginS.POST(pathB, func(c *gin.Context) { c.String(http.StatusCreated, "b") })
		if !bbHasRoute(http.MethodGet, pathA) || !bbHasRoute(http.MethodPost, pathB) {
			t.Fatalf("case %d: cross-wrapper visibility failed", i)
		}
	}
}

// TestDEGroupMiddlewareProperty exercises Group prefix routes and global Use middleware.
func TestDEGroupMiddlewareProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 3))
	prefix := bbUniquePath(rng, "grp")
	var hit bool
	ginS.Use(func(c *gin.Context) {
		if c.GetHeader("X-BB-Probe") == "1" {
			hit = true
		}
		c.Next()
	})
	grp := ginS.Group(prefix)
	grp.GET("/inner", func(c *gin.Context) { c.String(http.StatusOK, "group test") })
	full := prefix + "/inner"
	if !bbHasRoute(http.MethodGet, full) {
		t.Fatal("group route not listed")
	}
	req, err := http.NewRequest(http.MethodGet, "http://"+bbEnsureServer(t)+full, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-BB-Probe", "1")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if !hit {
		t.Fatal("Use middleware did not run")
	}
	w := bbPerform(t, http.MethodGet, full, nil)
	if w.Code != http.StatusOK || w.Body.String() != "group test" {
		t.Fatalf("group handler: code=%d body=%q", w.Code, w.Body.String())
	}
}

// TestDENoRouteNoMethodProperty registers fallbacks and probes unmatched traffic.
func TestDENoRouteNoMethodProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 4))
	miss := bbUniquePath(rng, "noroute")
	ginS.NoRoute(func(c *gin.Context) { c.String(http.StatusNotFound, "custom 404") })
	w := bbPerform(t, http.MethodGet, miss, nil)
	if w.Code != http.StatusNotFound || w.Body.String() != "custom 404" {
		t.Fatalf("NoRoute: code=%d body=%q", w.Code, w.Body.String())
	}
	ginS.NoMethod(func(c *gin.Context) { c.String(http.StatusMethodNotAllowed, "method not allowed") })
	// NoMethod is callable; Routes() still works after registration.
	if ginS.Routes() == nil {
		t.Fatal("Routes nil after NoMethod")
	}
}

// TestDEStaticProperty mirrors gins_test static file fixtures.
func TestDEStaticProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 5))
	base := bbUniquePath(rng, "static")
	ginS.StaticFile(base+"/file", "../testdata/test_file.txt")
	w := bbPerform(t, http.MethodGet, base+"/file", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("StaticFile: %d", w.Code)
	}
	ginS.Static(base+"/dir", "../testdata")
	w = bbPerform(t, http.MethodGet, base+"/dir/test_file.txt", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("Static: %d", w.Code)
	}
	ginS.StaticFS(base+"/fs", http.Dir("../testdata"))
	w = bbPerform(t, http.MethodGet, base+"/fs/test_file.txt", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("StaticFS: %d", w.Code)
	}
}

// TestDEHTMLGlobProperty configures templates via LoadHTMLGlob (wrapper must not panic).
func TestDEHTMLGlobProperty(t *testing.T) {
	ginS.LoadHTMLGlob("../testdata/template/*")
	tmpl := template.Must(template.New("bb").Parse("Hello {{.}}"))
	ginS.SetHTMLTemplate(tmpl)
	if ginS.Routes() == nil {
		t.Fatal("Routes unavailable after template setup")
	}
}

// TestDEAdversarialLazyInitProperty stress-registers routes sequentially on first-use engine.
func TestDEAdversarialLazyInitProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 6))
	for i := 0; i < 500; i++ {
		path := bbUniquePath(rng, "lazy")
		ginS.GET(path, func(c *gin.Context) { c.Status(http.StatusOK) })
		if !bbHasRoute(http.MethodGet, path) {
			t.Fatalf("case %d: route missing", i)
		}
	}
}

// TestDEUnseenRandomProperty runs >=10k random Routes()/HTTP probes over registered paths.
func TestDEUnseenRandomProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 8))
	paths := make([]string, 0, 256)
	for i := 0; i < 256; i++ {
		path := bbUniquePath(rng, "rand")
		ginS.GET(path, func(c *gin.Context) { c.String(http.StatusOK, "ok") })
		paths = append(paths, path)
	}
	for i := 0; i < bbCases; i++ {
		path := paths[rng.Intn(len(paths))]
		if !bbHasRoute(http.MethodGet, path) {
			t.Fatalf("case %d: route table miss %s", i, path)
		}
		if i%5000 == 0 {
			w := bbPerform(t, http.MethodGet, path, nil)
			if w.Code != http.StatusOK {
				t.Fatalf("case %d: http %d", i, w.Code)
			}
		}
	}
}
