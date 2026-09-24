// These tests regress HTTP code generation around OneOf request bodies and
// single-view ResultType responses. Client validation, union collection, and
// decode/init must all derive from the same effective transport body shape.
package codegen

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	"example.internal/apikit/v3/codegen"
	. "example.internal/apikit/v3/dsl"
	"example.internal/apikit/v3/expr"
	_ "example.internal/apikit/v3/http/codegen/testdata"
)



// renderClientCLISectionCode renders the requested client CLI section for a
// DSL and returns the generated code.
func renderClientCLISectionCode(t *testing.T, dsl func(), fileIndex, sectionIndex int) string {
	t.Helper()

	root := expr.RunDSL(t, dsl)
	plan := linkedHTTPPlanForRoot(t, root)
	fs := plan.ClientCLIFiles()

	return codegen.SectionCode(t, fs[fileIndex].SectionTemplates[sectionIndex])
}

// renderClientTypesCode renders the client type file for a single-service DSL.
func renderClientTypesCode(t *testing.T, dsl func()) string {
	t.Helper()

	root := expr.RunDSL(t, dsl)
	plan := linkedHTTPPlanForRoot(t, root)
	fs := plan.ClientTypeFiles()[0]

	var buf bytes.Buffer
	for _, s := range fs.SectionTemplates[1:] {
		require.NoError(t, s.Write(&buf))
	}

	return codegen.FormatTestCode(t, "package foo\n"+buf.String())
}

// renderClientDecodeCode renders the client decode section for a single-service
// DSL and returns the generated code.
func renderClientDecodeCode(t *testing.T, dsl func()) string {
	t.Helper()

	root := expr.RunDSL(t, dsl)
	plan := linkedHTTPPlanForRoot(t, root)
	fs := plan.ClientFiles()
	require.Len(t, fs, 2)

	sections := fs[1].SectionTemplates
	require.Greater(t, len(sections), 2)

	return codegen.SectionCode(t, sections[2])
}

// oneOfResultSingleViewDSL defines a ResultType whose only view drops the OneOf
// field. Client response code must therefore treat the transport body as the
// projected view, not the raw ResultType.
func oneOfResultSingleViewDSL() {
	animal := oneOfAnimalResultType("application/vnd.oneof-http-single-view.animal")

	Service("ServiceOneOfSingleView", func() {
		Method("MethodShowAnimal", func() {
			Payload(func() {
				Attribute("id", String)
				Required("id")
			})
			Result(animal)
			HTTP(func() {
				GET("/animals/{id}")
				Response(StatusOK)
			})
		})
	})
}

// oneOfResultCollectionSingleViewDSL defines a collection result whose
// transport body is fixed to a single view. The generated client collection
// code must project before collecting unions and body types.
func oneOfResultCollectionSingleViewDSL() {
	animal := oneOfAnimalResultType("application/vnd.oneof-http-collection-view.animal")

	Service("ServiceOneOfCollectionSingleView", func() {
		Method("MethodListAnimals", func() {
			Result(CollectionOf(animal), func() {
				View("default")
			})
			HTTP(func() {
				GET("/animals")
				Response(StatusOK)
			})
		})
	})
}

// oneOfAnimalResultType returns a ResultType whose default view excludes the
// OneOf details attribute. The omitted union is the regression target.
func oneOfAnimalResultType(mediaType string) *expr.ResultTypeExpr {
	var catDetails = Type("CatDetails", func() {
		Attribute("favorite_spot", String)
		Attribute("lives_left", Int)
		Required("favorite_spot", "lives_left")
	})
	var dogDetails = Type("DogDetails", func() {
		Attribute("favorite_park", String)
		Attribute("plays_fetch", Boolean)
		Required("favorite_park", "plays_fetch")
	})
	var birdDetails = Type("BirdDetails", func() {
		Attribute("can_fly", Boolean)
		Attribute("vocabulary_size", Int)
		Required("can_fly", "vocabulary_size")
	})
	var fishDetails = Type("FishDetails", func() {
		Attribute("water_type", String)
		Required("water_type")
	})

	return ResultType(mediaType, func() {
		TypeName("Animal")
		Attributes(func() {
			Attribute("name", String)
			OneOf("details", func() {
				Attribute("cat", catDetails)
				Attribute("dog", dogDetails)
				Attribute("bird", birdDetails)
				Attribute("fish", fishDetails)
			})
			Attribute("id", String)
		})
		View("default", func() {
			Attribute("name")
			Attribute("id")
		})
		Required("name", "id")
	})
}
