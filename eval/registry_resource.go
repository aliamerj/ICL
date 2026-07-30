package eval

// resourceRegistry tracks every declared resource by name, so later
// blocks can reference them (app_server.id) and so duplicate names
// across the whole file are caught immediately, not silently allowed.
type resourceRegistry struct {
	Instances map[string]*ResourceConfig
}

func (r *resourceRegistry) lookup(name string) (*ResourceConfig, bool) {
	cfg, ok := r.Instances[name]
	return cfg, ok
}

func (r *resourceRegistry) add(cfg *ResourceConfig) {
	r.Instances[cfg.Name] = cfg
}

func (r *resourceRegistry) has(name string) bool {
	_, ok := r.Instances[name]
	return ok
}
