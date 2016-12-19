package resolve

import (
	"errors"
	"reflect"
	"testing"

	"github.com/gonum/graph/simple"
	"github.com/stretchr/testify/assert"
)

// TODO: Ensure the inject.Injector has the values expected

func TestNode(t *testing.T) {
	g := simple.NewDirectedGraph(0, 0)
	nodeID := g.NewNodeID()

	// Test the happy path with no args
	n, err := newFuncNode(nodeID, func() {})
	assert.NoError(t, err, "error while making node")

	assert.Equal(t, len(n.provides), 0, "number of provided types should be 0")
	assert.Equal(t, len(n.requires), 0, "number of required types should be 0")

	// Ensure the node ID is set properly
	assert.Equal(t, n.ID(), nodeID, "Node ID does not match given ID")

	// Ensure we error on a non-function type
	n, err = newFuncNode(nodeID, 42)
	assert.Error(t, err, "no error while making node with invalid type")

	// Ensure we error on nil
	n, err = newFuncNode(nodeID, nil)
	assert.Error(t, err, "no error while making node with invalid type")

	// Ensure the types match when given one argument
	n, err = newFuncNode(nodeID, func(int) {})
	assert.NoError(t, err, "error while making node")

	assert.Equal(t, len(n.provides), 0, "number of provided types should be 0")
	assert.Equal(t, len(n.requires), 1, "number of required types should be 1")
	assert.Equal(t, n.requires[0].Kind(), reflect.Int, "required type should be Int")

	// Ensure the types match when given one return value
	n, err = newFuncNode(nodeID, func() int { return 0 })
	assert.NoError(t, err, "error while making node")

	assert.Equal(t, len(n.provides), 1, "number of provided types should be 1")
	assert.Equal(t, len(n.requires), 0, "number of required types should be 0")
	assert.Equal(t, n.provides[0].Kind(), reflect.Int, "provided type should be Int")
}

func TestAddNode(t *testing.T) {
	resolver := NewResolver()
	err := resolver.AddNode(func() {})
	assert.NoError(t, err, "error while adding node")

	// Ensure that adding a node to the resolver adds it to the internal graph.
	assert.Equal(t, len(resolver.graph.Nodes()), 1)

	// Ensure that adding an invalid type will return an error
	err = resolver.AddNode(1)
	assert.Error(t, err, "no error while adding invalid node type")

	// Ensure that adding a nil node will return an error
	err = resolver.AddNode(nil)
	assert.Error(t, err, "no error while adding nil node")

	// Ensure that adding a node which adds provided types will add them to
	// providedBy.
	err = resolver.AddNode(func() int { return 42 })
	assert.NoError(t, err, "error while adding node")

	// Ensure that adding a node with an overlapping provided type will error.
	err = resolver.AddNode(func() int { return 42 })
	assert.Error(t, err, "no error while adding duplicate provided type")

	// Ensure that error doesn't get added to the providedBy mapping
	err = resolver.AddNode(func() error { return nil })
	assert.NoError(t, err, "error while adding node")

	_, ok := resolver.providedBy[errorType]
	assert.False(t, ok, "error type should not be added to providedBy")

	// Ensure that no edges have been added to the graph.
	assert.Equal(t, len(resolver.graph.Edges()), 0)
}

func TestResolve(t *testing.T) {
	var (
		needsInt           = func(int) {}
		providesInt        = func() int { return 42 }
		returnsNilErr      = func() error { return nil }
		returnsNonNilError = func() error { return errors.New("Hello error") }
		cyclePartOne       = func(int) float32 { return 0.0 }
		cyclePartTwo       = func(float32) int { return 0 }
	)

	// Test empty dep chain
	r := NewResolver()
	_, err := r.Resolve()
	assert.NoError(t, err, "nothing in dep chain, so no error")

	// Test broken dependency chain
	r = NewResolver()
	err = r.AddNode(needsInt)
	assert.NoError(t, err)
	_, err = r.Resolve()
	assert.Error(t, err, "missing dependency, but no error")
	assert.Equal(t, err.Error(), "Missing dependencies: int")

	// Ensure when we have a valid dep chain, no error occurs.
	err = r.AddNode(providesInt)
	assert.NoError(t, err)
	_, err = r.Resolve()
	assert.NoError(t, err, "valid dep chain caused error")

	r = NewResolver()
	err = r.AddNode(returnsNilErr)
	assert.NoError(t, err)
	_, err = r.Resolve()
	assert.NoError(t, err, "returning nil error caused error")

	r = NewResolver()
	err = r.AddNode(returnsNonNilError)
	assert.NoError(t, err)
	_, err = r.Resolve()
	assert.Error(t, err, "returning non-nil error did not cause error")

	r = NewResolver()
	err = r.AddNode(cyclePartOne)
	assert.NoError(t, err)
	err = r.AddNode(cyclePartTwo)
	assert.NoError(t, err)
	_, err = r.Resolve()
	assert.Error(t, err, "cycle did not cause error")

	r = NewResolver()
	n, err := newFuncNode(r.graph.NewNodeID(), needsInt)
	assert.NoError(t, err)
	n.requires = nil // this will mess up the internal topo sort so we can get to the inject error
	r.graph.AddNode(n)
	_, err = r.Resolve()
	assert.Error(t, err, "injector did not cause error for requiring missing types")
}
