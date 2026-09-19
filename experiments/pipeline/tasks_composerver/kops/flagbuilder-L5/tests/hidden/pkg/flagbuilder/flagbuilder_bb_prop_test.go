// Package flagbuilder_test is a hidden black-box property suite for flagbuilder.
// Exported API only (api.md): BuildFlags, BuildFlagsList. Seed 20260919; >=10k cases.
//
// Contract (contract.md) -> property coverage table:
//
//	"durations, ints, nested structs, omit-empty" -> TestFlagbuilderKCMProperty
//	"kubelet tags including maps/slices" -> TestFlagbuilderKubeletProperty
//	"admission slices and repeat vs join" -> TestFlagbuilderAPIServerProperty
//	"quote only when joined-string form sees \"" -> TestFlagbuilderQuotingProperty
//	"sorted stable flag list" -> TestFlagbuilderSortedRandomProperty
package flagbuilder_test

import (
	"math/rand"
	"sort"
	"strings"
	"testing"
	"time"

	fb "example.internal/clustkit/pkg/flagbuilder"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const bbSeed = 20260919
const bbCases = 10000

type bbNested struct {
	Leaf int `flag:"leaf-count" flag-empty:"0"`
}

type bbKCM struct {
	Nested           bbNested          `flag:""`
	SyncPeriod       metav1.Duration   `flag:"horizontal-pod-autoscaler-sync-period"`
	NodeMonitorGrace metav1.Duration   `flag:"node-monitor-grace-period" flag-empty:"0s"`
	ClusterName      string            `flag:"cluster-name" flag-empty:""`
}

type bbKubelet struct {
	MaxPods    int               `flag:"max-pods" flag-empty:"0"`
	NodeLabels map[string]string `flag:"node-labels"`
	Register   *string           `flag:"register-with-taints" flag-include-empty:"true"`
	Skip       *string           `flag:"skipped-ptr"`
}

type bbAPIServer struct {
	Admission []string `flag:"admission-control,repeat"`
	Plugins   []string `flag:"plugins"`
	Secure    bool     `flag:"secure-port" flag-empty:"false"`
}

type bbQuote struct {
	Message string `flag:"message"`
}

type bbRandOpts struct {
	Alpha int               `flag:"alpha"`
	Beta  bool              `flag:"beta" flag-empty:"false"`
	Gamma map[string]string `flag:"gamma"`
	Delta metav1.Duration     `flag:"delta"`
	Eps   resource.Quantity   `flag:"eps"`
}

func bbFlagsSorted(argv []string) bool {
	cp := append([]string(nil), argv...)
	sort.Strings(cp)
	return strings.Join(cp, "\n") == strings.Join(argv, "\n")
}

func bbParseArgv(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return strings.Fields(s)
}

func TestFlagbuilderKCMProperty(t *testing.T) {
	opts := bbKCM{
		Nested:     bbNested{Leaf: 3},
		SyncPeriod: metav1.Duration{Duration: 0},
	}
	argv, err := fb.BuildFlagsList(&opts)
	if err != nil {
		t.Fatalf("BuildFlagsList: %v", err)
	}
	joined := strings.Join(argv, " ")
	if !strings.Contains(joined, "--horizontal-pod-autoscaler-sync-period=0s") {
		t.Fatalf("zero duration must be 0s: %v", argv)
	}
	if !strings.Contains(joined, "--leaf-count=3") {
		t.Fatalf("nested leaf missing: %v", argv)
	}
	if strings.Contains(joined, "node-monitor-grace-period") {
		t.Fatalf("flag-empty duration omitted: %v", argv)
	}
}

func TestFlagbuilderKubeletProperty(t *testing.T) {
	empty := ""
	opts := bbKubelet{
		MaxPods: 110,
		NodeLabels: map[string]string{
			"z": "last",
			"a": "first",
		},
		Register: &empty,
	}
	argv, err := fb.BuildFlagsList(&opts)
	if err != nil {
		t.Fatal(err)
	}
	var labels string
	for _, f := range argv {
		if strings.HasPrefix(f, "--node-labels=") {
			labels = strings.TrimPrefix(f, "--node-labels=")
		}
	}
	if labels != "a=first,z=last" {
		t.Fatalf("map keys sorted k=v: got %q", labels)
	}
	foundReg := false
	for _, f := range argv {
		if f == "--register-with-taints=" {
			foundReg = true
		}
	}
	if !foundReg {
		t.Fatalf("flag-include-empty *string must emit: %v", argv)
	}
	for _, f := range argv {
		if strings.HasPrefix(f, "--skipped-ptr") {
			t.Fatalf("nil *string omitted: %v", argv)
		}
	}
}

func TestFlagbuilderAPIServerProperty(t *testing.T) {
	repeat := bbAPIServer{
		Admission: []string{"A", "B"},
		Secure:    true,
	}
	argvR, err := fb.BuildFlagsList(&repeat)
	if err != nil {
		t.Fatal(err)
	}
	var repCount int
	for _, f := range argvR {
		if strings.HasPrefix(f, "--admission-control=") {
			repCount++
		}
	}
	if repCount != 2 {
		t.Fatalf("repeat emits one flag per element: %v", argvR)
	}
	join := bbAPIServer{Plugins: []string{"x", "y"}}
	argvJ, err := fb.BuildFlagsList(&join)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range argvJ {
		if strings.HasPrefix(f, "--plugins=") && f != "--plugins=x,y" {
			t.Fatalf("slice comma-joined: %v", argvJ)
		}
	}
}

func TestFlagbuilderQuotingProperty(t *testing.T) {
	opts := bbQuote{Message: "say \"hello\""}
	s, err := fb.BuildFlags(&opts)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(s, "\"") {
		t.Fatalf("BuildFlags must quote values containing double quote: %q", s)
	}
	argv, err := fb.BuildFlagsList(&opts)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range argv {
		if i := strings.Index(f, "="); i > 0 {
			val := f[i+1:]
			if len(val) >= 2 && val[0] == '"' && val[len(val)-1] == '"' {
				t.Fatalf("BuildFlagsList must not %%q-wrap values: %q", f)
			}
		}
	}
}

func TestFlagbuilderSortedRandomProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed))
	for i := 0; i < bbCases; i++ {
		m := map[string]string{}
		nm := rng.Intn(4)
		for j := 0; j < nm; j++ {
			k := string(rune('a'+j))
			m[k] = rngString(rng, 3)
		}
		opts := bbRandOpts{
			Alpha: rng.Intn(20),
			Beta:  rng.Intn(2) == 0,
			Gamma: m,
			Delta: metav1.Duration{Duration: time.Duration(rng.Intn(5)) * time.Second},
			Eps:   resource.MustParse("100m"),
		}
		argv, err := fb.BuildFlagsList(&opts)
		if err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		if !bbFlagsSorted(argv) {
			t.Fatalf("case %d: flags not sorted: %v", i, argv)
		}
		s, err := fb.BuildFlags(&opts)
		if err != nil {
			t.Fatalf("case %d BuildFlags: %v", i, err)
		}
		if len(bbParseArgv(s)) != len(argv) {
			t.Fatalf("case %d: BuildFlags field count mismatch", i)
		}
	}
}

func rngString(rng *rand.Rand, n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	var b strings.Builder
	for i := 0; i < n; i++ {
		b.WriteByte(chars[rng.Intn(len(chars))])
	}
	return b.String()
}
