package eval

type varRegistry struct {
	Instances map[string]*VarConfig
}

func (r *varRegistry) lookup(name string) (*VarConfig, bool) {
	cfg, ok := r.Instances[name]
	return cfg, ok
}

func (r *varRegistry) add(cfg *VarConfig) {
	r.Instances[cfg.Name] = cfg
}

func (r *varRegistry) has(name string) bool {
	_, ok := r.Instances[name]
	return ok
}
