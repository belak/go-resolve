package resolve

import (
	"errors"
	"reflect"
	"strings"

	"github.com/codegangsta/inject"
)

// It is more of a pain than it should be to get an interface type as just
// getting the type of a plain interface type will return nil. This method is
// explicitly mentioned in the example for reflect.TypeOf.
var errorType = reflect.TypeOf((*error)(nil)).Elem()

// TODO: Wrap errors

// EnsureValidFactory will return nil if a factory is valid and an
// error if the factory cannot be used.
func EnsureValidFactory(item interface{}) error {
	if item == nil {
		return errors.New("Factory cannot be nil")
	}

	// We need to ensure it's a function, otherwise the function calls to grab
	// the number of arguments and return values will panic.
	t := reflect.TypeOf(item)

	if t.Kind() != reflect.Func {
		return errors.New("Factory is not a func")
	}

	return nil
}

type funcNode struct {
	name string

	provides []reflect.Type
	requires []reflect.Type

	raw interface{}
}

func newFuncNode(name string, item interface{}) (*funcNode, error) {
	err := EnsureValidFactory(item)
	if err != nil {
		return nil, err
	}

	n := &funcNode{
		name: name,
		raw:  item,
	}

	// We've already ensured that this is a factory in
	// EnsureValidFactory, so we don't need to do it again here.
	t := reflect.TypeOf(item)

	// Grab all the provided args
	for i := 0; i < t.NumOut(); i++ {
		n.provides = append(n.provides, t.Out(i))
	}

	// Grab all the incoming args
	for i := 0; i < t.NumIn(); i++ {
		n.requires = append(n.requires, t.In(i))
	}

	return n, nil
}

// Resolver is a set of values which, when called in the proper order, contain
// all the requirements as return values of other functions.
type Resolver struct {
	nodes      []*funcNode
	names      map[string]bool
	providedBy map[reflect.Type]*funcNode
}

// NewResolver returns an empty resolve set which can be used for resolving
// function calls.
func NewResolver() *Resolver {
	return &Resolver{
		names:      make(map[string]bool),
		providedBy: make(map[reflect.Type]*funcNode),
	}
}

// AddNode adds a function to an internal graph of dependencies. The resolution
// will be done when Resolve is called.
func (r *Resolver) AddNode(name string, item interface{}) error {
	if r.names[name] {
		return errors.New("Name provided by multiple nodes")
	}

	n, err := newFuncNode(name, item)
	if err != nil {
		return err
	}

	// Ensure there are not overlapping provided types
	for _, t := range n.provides {
		// We don't care if multiple functions return errors, or even if
		// multiple errors are returned from a single constructor.
		if t.Implements(errorType) {
			continue
		}

		if _, ok := r.providedBy[t]; ok {
			return errors.New("Type provided by multiple constructors")
		}

		r.providedBy[t] = n
	}

	// Now that we have a valid node, we need to save it for later.
	r.nodes = append(r.nodes, n)
	r.names[name] = true

	return nil
}

// Resolve will walk the graph of constructor nodes, run the constructors in the
// order they need to be run, and return an injector with all the return values
// from these constructors. Any error returned by these constructors will be
// returned by Resolve if the constructor returns them and is non nil. Note that
// because this requires a topological sort every time this is run, it is
// recommended to not use this often. Additionally, all nodes must be added
// before this method is called.
func (r *Resolver) Resolve() (inject.Injector, error) {
	order, err := r.getOrder()
	if err != nil {
		return nil, err
	}

	return createInjector(order)
}

func (r *Resolver) getOrder() ([]*funcNode, error) {
	nodeDependencies := map[*funcNode]map[*funcNode]bool{}
	missingDeps := map[reflect.Type]bool{}

	// Loop over all nodes and add edges for all requirements
	for _, n := range r.nodes {
		nodeDependencies[n] = make(map[*funcNode]bool)

		for _, t := range n.requires {
			depNode, ok := r.providedBy[t]
			if !ok {
				missingDeps[t] = true
				continue
			}

			nodeDependencies[n][depNode] = true
		}
	}

	if len(missingDeps) > 0 {
		missingDepStrs := []string{}
		for dep := range missingDeps {
			missingDepStrs = append(missingDepStrs, dep.String())
		}
		return nil, errors.New("Missing dependencies: " + strings.Join(missingDepStrs, ", "))
	}

	var order []*funcNode

	// Loop through nodeDependencies as long as there are any left
	for len(nodeDependencies) > 0 {
		var ready []*funcNode

		for node, deps := range nodeDependencies {
			if len(deps) > 0 {
				continue
			}

			ready = append(ready, node)
		}

		// If there are no ready nodes, we have a circular dependency
		if len(ready) == 0 {
			// TODO: Display the nodes in the cycle
			return nil, errors.New("Circular dependency found")
		}

		for _, node := range ready {
			// Remove the node from what's left to handle.
			delete(nodeDependencies, node)

			// Add the node to the returned order
			order = append(order, node)

			// Remove this dependency from other listed nodes.
			for _, deps := range nodeDependencies {
				delete(deps, node)
			}
		}
	}

	return order, nil
}

func createInjector(order []*funcNode) (inject.Injector, error) {
	// Create a new injector for returning
	injector := inject.New()

	// For each node, we need to call it, then add the returned values to the
	// injector.
	for _, n := range order {
		vals, err := injector.Invoke(n.raw)
		if err != nil {
			// Note that this shouldn't be possible to hit because we already
			// ensured there are no missing deps above.
			return nil, err
		}

		for _, v := range vals {
			// If we got a non-nil error, we need to return it.
			if err, ok := v.Interface().(error); ok && err != nil {
				return nil, err
			}

			// Add any non-error types to the injector.
			injector.Set(v.Type(), v)
		}
	}

	return injector, nil
}
