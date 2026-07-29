package eval

// resourceRegistry tracks every declared resource by name, so later
// blocks can reference them (app_server.id) and so duplicate names
// across the whole file are caught immediately, not silently allowed.
type resourceRegistry struct {
	instances map[string]*ResourceConfig
}

func (r *resourceRegistry) Lookup(name string) (*ResourceConfig, bool) {
	cfg, ok := r.instances[name]
	return cfg, ok
}

func (r *resourceRegistry) Add(cfg *ResourceConfig) {
	r.instances[cfg.Name] = cfg
}

func (r *resourceRegistry) Has(name string) bool {
	_, ok := r.instances[name]
	return ok
}
