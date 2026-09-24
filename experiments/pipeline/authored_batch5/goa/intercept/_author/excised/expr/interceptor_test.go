package expr

import (
	"testing"
)


// Test helpers (at the end of the file)
func withWritePayload(t *testing.T, attrs *NamedAttributeExpr) func(*InterceptorExpr) {
	t.Helper()
	return func(i *InterceptorExpr) {
		i.WritePayload = &AttributeExpr{Type: &Object{attrs}}
	}
}

func withResultBases(t *testing.T, bases ...DataType) func(*MethodExpr) {
	t.Helper()
	return func(m *MethodExpr) {
		if m.Result == nil {
			m.Result = &AttributeExpr{Type: &Object{}}
		}
		m.Result.Bases = bases
	}
}

func makeInterceptor(t *testing.T, opts ...func(*InterceptorExpr)) *InterceptorExpr {
	t.Helper()
	i := &InterceptorExpr{Name: "test-interceptor"}
	for _, opt := range opts {
		opt(i)
	}
	return i
}

// Helper functions need to be updated to handle empty attributes properly
func withReadPayload(t *testing.T, attrs *NamedAttributeExpr) func(*InterceptorExpr) {
	t.Helper()
	return func(i *InterceptorExpr) {
		i.ReadPayload = &AttributeExpr{Type: &Object{attrs}}
	}
}

func withReadResult(t *testing.T, attrs *NamedAttributeExpr) func(*InterceptorExpr) {
	t.Helper()
	return func(i *InterceptorExpr) {
		i.ReadResult = &AttributeExpr{Type: &Object{attrs}}
	}
}

func withReadStreamingPayload(t *testing.T, attrs *NamedAttributeExpr) func(*InterceptorExpr) {
	t.Helper()
	return func(i *InterceptorExpr) {
		i.ReadStreamingPayload = &AttributeExpr{Type: &Object{attrs}}
	}
}

func withReadStreamingResult(t *testing.T, attrs *NamedAttributeExpr) func(*InterceptorExpr) {
	t.Helper()
	return func(i *InterceptorExpr) {
		i.ReadStreamingResult = &AttributeExpr{Type: &Object{attrs}}
	}
}

func makeMethod(t *testing.T, opts ...func(*MethodExpr)) *MethodExpr {
	t.Helper()
	m := &MethodExpr{}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

func withPayload(t *testing.T, attrs *NamedAttributeExpr) func(*MethodExpr) {
	t.Helper()
	return func(m *MethodExpr) {
		m.Payload = &AttributeExpr{Type: &Object{attrs}}
	}
}

func withPayloadBases(t *testing.T, bases ...DataType) func(*MethodExpr) {
	t.Helper()
	return func(m *MethodExpr) {
		if m.Payload == nil {
			m.Payload = &AttributeExpr{Type: &Object{}}
		}
		m.Payload.Bases = bases
	}
}

func withResult(t *testing.T, attrs *NamedAttributeExpr) func(*MethodExpr) {
	t.Helper()
	return func(m *MethodExpr) {
		m.Result = &AttributeExpr{Type: &Object{attrs}}
	}
}

func namedAttr(t *testing.T, name string) *NamedAttributeExpr {
	t.Helper()
	return &NamedAttributeExpr{
		Name: name,
		Attribute: &AttributeExpr{
			Type: String,
		},
	}
}
