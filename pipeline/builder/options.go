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

// WithTimeoutOverride overrides every stage's default 72 hour timeout
func WithTimeoutOverride(hours int) OptFunc {
	return func(b *Builder) {
		b.timeoutHours = hours
	}
}

// WithAccountOverride lets you override an account with a different account
func WithAccountOverride(accounts map[string]string) OptFunc {
	return func(b *Builder) {
		b.overrideAccounts = accounts
	}
}

// WithKubecostData
func WithKubecostData(data map[string][]byte) OptFunc {
	return func(b *Builder) {
		b.kubecostData = data
	}
}
