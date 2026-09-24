package openapi

// mergeItems is an internal datatype used to merge two schemas.
type mergeItems []struct {
	a, b   any
	needed bool
}

// Merge does a two level deep merge of other into s.
func (s *Schema) Merge(other *Schema) {
	panic("excised: Schema.Merge")
}

func (s *Schema) createMergeItems(other *Schema) mergeItems {
	panic("excised: Schema.createMergeItems")
}
