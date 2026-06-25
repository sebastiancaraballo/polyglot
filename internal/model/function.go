package model

// Function is a communicative function in the language-agnostic curriculum spine:
// something a learner can do with language (greet, count, ask for directions),
// graded with a CEFR level. Functions are shared across all language pairs; each
// per-language lesson references them to provide a concrete realization (the
// cultural "skin" over the universal "spine").
type Function struct {
	ID          string
	CEFR        CEFR
	Description string // authored in-house; not copied from external can-do catalogs
}

// FunctionCatalog maps a function ID to its definition.
type FunctionCatalog map[string]Function

// Lookup returns the function for id and whether it exists.
func (c FunctionCatalog) Lookup(id string) (Function, bool) {
	f, ok := c[id]
	return f, ok
}
