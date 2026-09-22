package expr_test

import (
	_ "testing"

	. "example.internal/apikit/v3/dsl"
	_ "example.internal/apikit/v3/expr"
)


var stringErrorResponseWithHeadersDSL = func() {
	Service("StringErrorResponseWithHeaders", func() {
		Method("Method", func() {
			Error("error", String)
			HTTP(func() {
				POST("/")
				Response("error", func() {
					Header("Location")
				})
			})
		})
	})
}

var objectErrorResponseWithHeadersDSL = func() {
	Service("ObjectErrorResponseWithHeaders", func() {
		Method("Method", func() {
			Error("error", func() {
				Attribute("foo", String)
			})
			HTTP(func() {
				POST("/")
				Response("error", func() {
					Header("foo:Location")
				})
			})
		})
	})
}

var implicitObjectErrorResponseWithHeadersDSL = func() {
	Service("ArrayObjectErrorResponseWithHeaders", func() {
		Method("Method", func() {
			Error("error", func() {
				Attribute("foo", func() {
					Attribute("bar", String)
					Attribute("baz", String)
				})
			})
			HTTP(func() {
				POST("/")
				Response("error", func() {
					Header("foo:Location")
				})
			})
		})
	})
}

var arrayObjectErrorResponseWithHeadersDSL = func() {
	var Obj = Type("Obj", func() {
		Attribute("foo", String)
	})
	Service("ArrayObjectErrorResponseWithHeaders", func() {
		Method("Method", func() {
			Error("error", ArrayOf(Obj))
			HTTP(func() {
				POST("/")
				Response("error", func() {
					Header("foo:Location")
				})
			})
		})
	})
}

var mapErrorTypeResponseWithHeadersDSL = func() {
	Service("MapErrorTypeResponseWithHeaders", func() {
		Method("Method", func() {
			Error("error", MapOf(String, Int))
			HTTP(func() {
				POST("/")
				Response("error", func() {
					Header("Location")
				})
			})
		})
	})
}

var arrayErrorResponseWithHeadersDSL = func() {
	Service("ArrayErrorResponseWithHeaders", func() {
		Method("Method", func() {
			Error("error", func() {
				Attribute("foo", ArrayOf(String))
			})
			HTTP(func() {
				POST("/")
				Response("error", func() {
					Header("foo:Location")
				})
			})
		})
	})
}

var mapErrorResponseWithHeadersDSL = func() {
	Service("MapErrorResponseWithHeaders", func() {
		Method("Method", func() {
			Error("error", func() {
				Attribute("foo", MapOf(String, String))
			})
			HTTP(func() {
				POST("/")
				Response("error", func() {
					Header("foo:Location")
				})
			})
		})
	})
}

var missingHeaderErrorAttributeDSL = func() {
	Service("MissingHeaderErrorAttribute", func() {
		Method("Method", func() {
			Error("error")
			HTTP(func() {
				POST("/")
				Response("error", func() {
					Header("bar")
				})
			})
		})
	})
}
