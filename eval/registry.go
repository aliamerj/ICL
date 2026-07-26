package eval

// Registry is the single, project-wide table of everything that can be
// referenced from anywhere in a file. Each reference *kind* gets its own
// typed sub-registry 
type Registry struct {
	providers *providerRegistry
	// Resources *ResourceRegistry // added when `resource` is built
	// Locals    *LocalRegistry    // added when `let` is built
}
