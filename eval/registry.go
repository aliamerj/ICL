package eval

// Registry is the single, project-wide table of everything that can be
// referenced from anywhere in a file. Each reference *kind* gets its own
// typed sub-registry
type Registry struct {
	Providers *providerRegistry
	Resources *resourceRegistry // added when `resource` is built
	Vars      *varRegistry
	Outputs   *outputRegistry
}
