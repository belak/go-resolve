package resolve

import (
	"errors"
	"reflect"

	"github.com/codegangsta/inject"
	"github.com/gonum/graph"
	"github.com/gonum/graph/simple"
	"github.com/gonum/graph/topo"
)

// It is more of a pain than it should be to get an interface type as just
// getting the type of a plain interface type will return nil. This method is
// explicitly mentioned in the example for reflect.TypeOf.
var errorType = reflect.TypeOf((*error)(nil)).Elem()

// TODO: Add method of simply adding dependency nodes without requiring the use
// of a constructor. This is meant to be used when building frameworks.

// TODO: Wrap errors

type node struct {
	nodeID int

	provides []reflect.Type
	requires []reflect.Type

	raw interface{}
}

func newNode(nodeID int, item interface{}) (*node, error) {
	if item == nil {
		return nil, errors.New("Must not pass a nil item into newNode")
	}

	n := &node{
		nodeID: nodeID,
		raw:    item,
	}

	// We need to ensure it's a function, otherwise the function calls to grab
	// the number of arguments and return values will panic.
	t := reflect.TypeOf(item)
	if t.Kind() != reflect.Func {
		return nil, errors.New("Not a func")
	}

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

func (n *node) ID() int {
	return n.nodeID
}

// Resolver is a set of values which, when called in the proper order, contain
// all the requirements as return values of other functions.
type Resolver struct {
	graph *simple.DirectedGraph

	providedBy map[reflect.Type]*node
}

// NewResolver returns an empty resolve set which can be used for resolving
// function calls.
func NewResolver() *Resolver {
	return &Resolver{
		graph: simple.NewDirectedGraph(0, 0),

		providedBy: make(map[reflect.Type]*node),
	}
}

// AddNode adds a function to an internal graph of dependencies. The resolution
// will be done when Resolve is called.
func (r *Resolver) AddNode(item interface{}) error {
	n, err := newNode(r.graph.NewNodeID(), item)
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

	// Now that we have a valid node, we need to add it to the graph.
	r.graph.AddNode(n)

	return nil
}

// Resolve will walk the graph of constructor nodes, run the constructors in the
// order they need to be run, and return an injector with all the return values
// from these constructors. Any error returned by these constructors will be
// returned by Resolve if the constructor returns them and is non nil. Note that
// because this requires a topological sort every time this is run, it is
// recommended to not use this often.
func (r *Resolver) Resolve() (inject.Injector, error) {
	g := simple.NewDirectedGraph(0, 0)

	// Copy the current node graph into a new one, in case we want to do this
	// later, so the edges don't overlap.
	graph.Copy(g, r.graph)

	// Loop over all nodes and add edges for all requirements
	for _, rawNode := range g.Nodes() {
		// We need our original node type. Because this is controlled
		// internally, we don't need to check if this type inference works.
		n := rawNode.(*node)

		for _, t := range n.requires {
			depNode, ok := r.providedBy[t]
			if !ok {
				return nil, errors.New("Missing dependency")
			}

			// Each requirement is defined as an edge from the dependency to the
			// dependent nodes. This will cause a topological sort to return the
			// order in which nodes should be loaded.
			g.SetEdge(simple.Edge{
				F: depNode,
				T: n,
				W: 1,
			})
		}
	}

	// Now that the full graph with edges is finished, we run a sort and start
	// working through the dependency nodes.
	order, err := topo.Sort(g)
	if err != nil {
		return nil, err
	}

	// Create a new injector for returning
	injector := inject.New()

	// For each node, we need to call it, then add the returned values to the
	// injector.
	for _, rawNode := range order {
		n := rawNode.(*node)
		vals, err := injector.Invoke(n.raw)
		if err != nil {
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