package eval

// Environment is a scope: local bindings (values) plus a shared pointer
// back to the one Registry for the whole program. Child scopes (future
// `for` bodies, nested blocks) get their own `values` but always share
// the same Registry — a for-loop shouldn't spawn its own copy of every
// provider/resource in the file, it just needs its own loop variable.
type environment struct {
	parent   *environment
	values   map[string]Value
	registry *Registry
}

func newEnv() *environment {
	return &environment{
		values: make(map[string]Value),
		registry: &Registry{
			providers: &providerRegistry{
				instances: make(map[string]*ProviderConfig),
			},
		},
	}
}

func (e *environment) child() *environment {
	return &environment{
		parent: e,
		values: map[string]Value{},
	}
}

func (e *environment) get(name string) (Value, bool) {
	if v, ok := e.values[name]; ok {
		return v, true
	}
	if e.parent != nil {
		return e.parent.get(name)
	}
	return Value{}, false
}

func (e *environment) set(name string, v Value) {
	e.values[name] = v
}
