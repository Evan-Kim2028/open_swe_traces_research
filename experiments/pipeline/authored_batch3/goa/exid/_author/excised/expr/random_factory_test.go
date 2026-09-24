// This file verifies that immutable example configuration creates independent
// mutable value streams for each code generation run.
package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"example.internal/apikit/v3/expr"
)

type (
	// customRandomizerFactory exercises the public factory contract without
	// depending on Apikit's built-in factory implementations.
	customRandomizerFactory struct {
		seed string
	}

	recordingRandomizerFactory struct {
		identities *[]expr.ExampleIdentity
	}
)

// NewRandomizer creates an independent seeded stream for identity.
func (f customRandomizerFactory) NewRandomizer(identity expr.ExampleIdentity) expr.Randomizer {
	if identity.Seed() == "" {
		panic("custom randomizer received an empty identity")
	}
	return expr.NewFakerRandomizerFactory(f.seed).NewRandomizer(identity)
}

// NewRandomizer records the exact owner selected by example traversal and
// delegates value generation to Apikit's deterministic factory.
func (f recordingRandomizerFactory) NewRandomizer(identity expr.ExampleIdentity) expr.Randomizer {
	*f.identities = append(*f.identities, identity)
	return expr.NewDeterministicRandomizerFactory().NewRandomizer(identity)
}

func TestRandomizerFactoriesCreateIndependentStreams(t *testing.T) {
	cases := []struct {
		Name    string
		Factory expr.RandomizerFactory
	}{
		{"faker", expr.NewFakerRandomizerFactory("seed")},
		{"deterministic", expr.NewDeterministicRandomizerFactory()},
		{"custom", customRandomizerFactory{seed: "seed"}},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			identity := expr.MethodPayloadExampleIdentity(exampleMethod("service", "method"))
			first := expr.NewExampleGenerator(c.Factory).At(identity)
			second := expr.NewExampleGenerator(c.Factory).At(identity)

			require.NotSame(t, first, second)
			require.Equal(t, first.String(), second.String())
			require.Equal(t, first.Int(), second.Int())
		})
	}
}

// TestReleasedStandaloneRandomizers checks the released concrete randomizer
// types, seed, and repeatable values.
func TestReleasedStandaloneRandomizers(t *testing.T) {
	faker := expr.NewFakerRandomizer("seed")
	concrete, ok := faker.(*expr.FakerRandomizer)
	require.True(t, ok)
	require.Equal(t, "seed", concrete.Seed)
	require.Equal(t, expr.NewFakerRandomizer("seed").String(), faker.String())

	deterministic := expr.NewDeterministicRandomizer()
	_, ok = deterministic.(*expr.DeterministicRandomizer)
	require.True(t, ok)
	require.Equal(t, "abc123", deterministic.String())
}

func TestRandomizerFactoriesPreserveDerivedExampleStability(t *testing.T) {
	factory := expr.NewFakerRandomizerFactory("seed")
	identity := expr.MethodPayloadExampleIdentity(exampleMethod("service", "method"))
	first := expr.NewExampleGenerator(factory).At(identity)
	second := expr.NewExampleGenerator(factory).At(identity)

	require.Equal(t, first.Member("payload").String(), second.Member("payload").String())
	require.Equal(t, first.ArrayElement(0).Int(), second.ArrayElement(0).Int())
}









func TestDisabledExampleGeneratorSuppressesAuthoredExamples(t *testing.T) {
	attribute := &expr.AttributeExpr{
		Type:         expr.String,
		UserExamples: []*expr.ExampleExpr{{Value: "authored"}},
	}

	require.Nil(t, attribute.Example(&expr.ExampleGenerator{}))
}

func exampleMethod(service, method string) *expr.MethodExpr {
	svc := &expr.ServiceExpr{Name: service}
	return &expr.MethodExpr{Name: method, Service: svc}
}
