package eval

// Environment holds named values in scope — resources, data lookups,
// locals, inputs. Flat map today; parent-scope chaining is the natural
// extension point once `for`/nested scopes exist, so the shape is
// deliberately ready for that without a rewrite.
type Environment struct {
	parent *Environment
	values map[string]Value
}

func NewEnv() *Environment {
	return &Environment{
		values: make(map[string]Value, 0),
	}
}

func (e *Environment) Child() *Environment {
	return &Environment{
		parent: e,
		values: make(map[string]Value, 0),
	}
}

func (e *Environment) Get(name string) (Value, bool) {
	if v, ok := e.values[name]; ok {
		return v, true
	}
	if e.parent != nil {
		return e.parent.Get(name)
	}
	return Value{}, false
}

func (e *Environment) Set(name string, v Value) {
	e.values[name] = v
}
