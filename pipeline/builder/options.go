package builder

// OptFunc is used to assign configuration values to a pipeline builder
type OptFunc func(b *Builder)

// WithLinear makes the builder assign references automatically for every
// stage within a pipeline config
func WithLinear(l bool) OptFunc {
	return func(b *Builder) {
		b.isLinear = l
	}
}
