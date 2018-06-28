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

// WithBasePath assigns the base path for the builder to use when given
// relatively pathed files for manifests
func WithBasePath(basePath string) OptFunc {
	return func(b *Builder) {
		b.basePath = basePath
	}
}

// WithV2Provider creates a json message adhering to the V2 Spinnaker Pipeline Spec
func WithV2Provider(v bool) OptFunc {
	return func(b *Builder) {
		b.v2Provider = v
	}
}
