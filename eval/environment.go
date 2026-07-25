package eval

// environment holds named values in scope - resources, data lookups,
// locals, inputs. Flat map today; parent-scope chaining is the natural
// extension point once `for`/nested scopes exist, so the shape is
// deliberately ready for that without a rewrite.
type environment struct {
	parent *environment
	values map[string]Value
}

func newEnv() *environment {
	return &environment{
		values: make(map[string]Value),
	}
}

func (e *environment) child() *environment {
	return &environment{
		parent: e,
		values: make(map[string]Value),
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
