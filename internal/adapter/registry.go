package adapter

import "fmt"

// Registry holds adapters in registration order.
type Registry struct {
	byName map[string]Adapter
	order  []string
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{byName: map[string]Adapter{}}
}

// Register adds a to the registry. Panics on duplicate name.
func (r *Registry) Register(a Adapter) {
	if _, exists := r.byName[a.Name()]; exists {
		panic(fmt.Sprintf("adapter %q already registered", a.Name()))
	}
	r.byName[a.Name()] = a
	r.order = append(r.order, a.Name())
}

// Names returns adapter names in registration order.
func (r *Registry) Names() []string {
	out := make([]string, len(r.order))
	copy(out, r.order)
	return out
}

// DetectedNames returns names of adapters whose Detect() returns true.
func (r *Registry) DetectedNames() []string {
	var out []string
	for _, n := range r.order {
		if r.byName[n].Detect() {
			out = append(out, n)
		}
	}
	return out
}

// Get returns the adapter for name and whether it was found.
func (r *Registry) Get(name string) (Adapter, bool) {
	a, ok := r.byName[name]
	return a, ok
}

// All returns adapters in registration order.
func (r *Registry) All() []Adapter {
	out := make([]Adapter, 0, len(r.order))
	for _, n := range r.order {
		out = append(out, r.byName[n])
	}
	return out
}

// Default is the process-global registry used by main(). Tests should
// construct their own Registry via NewRegistry to avoid global state.
var Default = NewRegistry()
