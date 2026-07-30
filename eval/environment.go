package eval

// Environment is a scope: local bindings (values) plus a shared pointer
// back to the one Registry for the whole program. Child scopes (future
// `for` bodies, nested blocks) get their own `values` but always share
// the same Registry — a for-loop shouldn't spawn its own copy of every
// provider/resource in the file, it just needs its own loop variable.
type Environment struct {
	Parent   *Environment
	Values   map[string]Value
	Registry *Registry
}

func NewEnv() *Environment {
	return &Environment{
		Values: make(map[string]Value),
		Registry: &Registry{
			Providers: &providerRegistry{
				Instances: make(map[string]*ProviderConfig),
			},
			Resources: &resourceRegistry{
				Instances: make(map[string]*ResourceConfig),
			},
		},
	}
}

func (e *Environment) child() *Environment {
	return &Environment{
		Parent: e,
		Values: map[string]Value{},
	}
}

func (e *Environment) get(name string) (Value, bool) {
	if v, ok := e.Values[name]; ok {
		return v, true
	}
	if e.Parent != nil {
		return e.Parent.get(name)
	}
	return Value{}, false
}

func (e *Environment) set(name string, v Value) {
	e.Values[name] = v
}
