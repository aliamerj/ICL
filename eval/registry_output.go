package eval

type outputRegistry struct {
	Instances map[string]*OutputConfig
}

func (r *outputRegistry) lookup(name string) (*OutputConfig, bool) {
	cfg, ok := r.Instances[name]
	return cfg, ok
}

func (r *outputRegistry) add(cfg *OutputConfig) {
	r.Instances[cfg.Name] = cfg
}

func (r *outputRegistry) has(name string) bool {
	_, ok := r.Instances[name]
	return ok
}
