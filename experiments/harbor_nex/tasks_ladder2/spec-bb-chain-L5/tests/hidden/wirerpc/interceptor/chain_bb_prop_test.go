package interceptor

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	"example.internal/kvstore/v2/wirerpc"
)

const chainSeed = 20260918
const chainCases = 10000

func chainMark(name string, log *[]string) RPCInterceptor {
	return NewRPCInterceptor(name, func(next RPCInterceptorFunc) RPCInterceptorFunc {
		return func(target string, req *tikvrpc.Request) (*tikvrpc.Response, error) {
			*log = append(*log, name)
			return next(target, req)
		}
	})
}

func TestInterceptorChainOrder(t *testing.T) {
	rng := rand.New(rand.NewSource(chainSeed))
	for i := 0; i < chainCases; i++ {
		n := rng.Intn(6) + 1
		names := make([]string, n)
		for j := 0; j < n; j++ {
			names[j] = fmt.Sprintf("n%d-%d", i, j)
		}
		chain := NewRPCInterceptorChain()
		var log []string
		for _, name := range names {
			chain.Link(chainMark(name, &log))
		}
		if chain.Len() != n {
			t.Fatalf("case %d Len=%d want %d", i, chain.Len(), n)
		}
		_, _ = chain.Wrap(func(target string, req *tikvrpc.Request) (*tikvrpc.Response, error) {
			log = append(log, "base")
			return nil, nil
		})("t", nil)
		if len(log) != n+1 {
			t.Fatalf("case %d log %v", i, log)
		}
		for j, name := range names {
			if log[j] != name {
				t.Fatalf("case %d order: got %v want prefix %v", i, log, names)
			}
		}
		if log[n] != "base" {
			t.Fatalf("case %d missing base: %v", i, log)
		}
	}
}

func TestInterceptorChainDedupAndCompose(t *testing.T) {
	var log []string
	a := chainMark("dup", &log)
	b := chainMark("dup", &log)
	chain := NewRPCInterceptorChain().Link(a).Link(b)
	if chain.Len() != 1 {
		t.Fatalf("same name kept twice: Len=%d", chain.Len())
	}
	first := chainMark("first", &log)
	second := chainMark("second", &log)
	composed := ChainRPCInterceptors(first, second)
	log = log[:0]
	_, _ = composed.Wrap(func(target string, req *tikvrpc.Request) (*tikvrpc.Response, error) {
		return nil, nil
	})("t", nil)
	if len(log) != 2 || log[0] != "first" || log[1] != "second" {
		t.Fatalf("compose order %v", log)
	}
}

func TestInterceptorContractExamples(t *testing.T) {
	var log []string
	it := NewRPCInterceptorChain().
		Link(chainMark("first", &log)).
		Link(chainMark("second", &log))
	_, _ = it.Wrap(func(target string, req *tikvrpc.Request) (*tikvrpc.Response, error) {
		log = append(log, "base")
		return nil, nil
	})("t", nil)
	if len(log) != 3 || log[0] != "first" || log[1] != "second" || log[2] != "base" {
		t.Fatalf("contract example order %v", log)
	}
	ctx := WithRPCInterceptor(context.Background(), chainMark("ctx", &log))
	got := GetRPCInterceptorFromCtx(ctx)
	if got == nil {
		t.Fatal("context binding returned nil")
	}
	if GetRPCInterceptorFromCtx(context.Background()) != nil {
		t.Fatal("empty context must yield nil interceptor")
	}
}

func TestInterceptorUnmentionedRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(chainSeed + 1))
	for i := 0; i < chainCases; i++ {
		a := fmt.Sprintf("u%x", rng.Uint64())
		b := fmt.Sprintf("v%x", rng.Uint64())
		if a == "first" || b == "second" || a == "dup" || b == "dup" {
			continue
		}
		var log []string
		it := ChainRPCInterceptors(chainMark(a, &log), chainMark(b, &log))
		_, _ = it.Wrap(func(target string, req *tikvrpc.Request) (*tikvrpc.Response, error) {
			return nil, nil
		})("t", nil)
		if len(log) != 2 || log[0] != a || log[1] != b {
			t.Fatalf("unseen %d: got %v want %s,%s", i, log, a, b)
		}
	}
}
