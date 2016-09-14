package gcss

// context represents a context of the parsing process.
type context struct {
	vars   map[string]*variable
	mixins map[string]*mixinDeclaration
}

// newContext creates and returns a context.
func newContext() *context {
	return &context{
		vars:   make(map[string]*variable),
		mixins: make(map[string]*mixinDeclaration),
	}
}
