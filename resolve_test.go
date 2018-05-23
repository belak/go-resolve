package resolve

import (
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TODO: Ensure the inject.Injector has the values expected

func TestNode(t *testing.T) {
	// Test the happy path with no args
	n, err := newFuncNode("A", func() {})
	assert.NoError(t, err, "unexpected error while making node")

	assert.Equal(t, len(n.provides), 0, "number of provided types should be 0")
	assert.Equal(t, len(n.requires), 0, "number of required types should be 0")

	// Ensure we error on a non-function type
	_, err = newFuncNode("A", 42)
	assert.Error(t, err, "no error while making node with invalid type")

	// Ensure we error on nil
	_, err = newFuncNode("A", nil)
	assert.Error(t, err, "no error while making node with invalid type")

	// Ensure the types match when given one argument
	n, err = newFuncNode("A", func(int) {})
	assert.NoError(t, err, "unexpected error while making node")

	assert.Equal(t, len(n.provides), 0, "number of provided types should be 0")
	assert.Equal(t, len(n.requires), 1, "number of required types should be 1")
	assert.Equal(t, n.requires[0].Kind(), reflect.Int, "required type should be Int")

	// Ensure the types match when given one return value
	n, err = newFuncNode("A", func() int { return 0 })
	assert.NoError(t, err, "unexpected error while making node")

	assert.Equal(t, len(n.provides), 1, "number of provided types should be 1")
	assert.Equal(t, len(n.requires), 0, "number of required types should be 0")
	assert.Equal(t, n.provides[0].Kind(), reflect.Int, "provided type should be Int")
}

func TestAddNode(t *testing.T) {
	resolver := NewResolver()
	err := resolver.AddNode("A", func() {})
	assert.NoError(t, err, "unexpected error while adding node")

	err = resolver.AddNode("A", func() {})
	assert.Error(t, err, "no error with duplicate node name")

	// Ensure that adding a node to the resolver adds it to the internal node
	// list.
	assert.Equal(t, len(resolver.nodes), 1)

	// Ensure that adding an invalid type will return an error
	err = resolver.AddNode("B", 1)
	assert.Error(t, err, "no error while adding invalid node type")

	// Ensure that adding a nil node will return an error
	err = resolver.AddNode("C", nil)
	assert.Error(t, err, "no error while adding nil node")

	// Ensure that adding a node which adds provided types will add them to
	// providedBy.
	err = resolver.AddNode("D", func() int { return 42 })
	assert.NoError(t, err, "unexpected error while adding node")

	// Ensure that adding a node with an overlapping provided type will error.
	err = resolver.AddNode("E", func() int { return 42 })
	assert.Error(t, err, "no error while adding duplicate provided type")

	// Ensure that error doesn't get added to the providedBy mapping
	err = resolver.AddNode("F", func() error { return nil })
	assert.NoError(t, err, "unexpected error while adding node")

	_, ok := resolver.providedBy[errorType]
	assert.False(t, ok, "error type should not be added to providedBy")
}

func TestResolve(t *testing.T) {
	var (
		needsInt           = func(int) {}
		needsTestingT      = func(*testing.T) {}
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

	// Test broken dependency chains
	r = NewResolver()
	err = r.AddNode("A", needsTestingT)
	assert.NoError(t, err)
	_, err = r.Resolve()
	assert.Error(t, err, "missing dependency, but no error")
	assert.Equal(t, err.Error(), "Missing dependencies: *testing.T")

	r = NewResolver()
	err = r.AddNode("A", needsInt)
	assert.NoError(t, err)
	_, err = r.Resolve()
	assert.Error(t, err, "missing dependency, but no error")
	assert.Equal(t, err.Error(), "Missing dependencies: int")

	// Ensure when we have a valid dep chain, no error occurs.
	err = r.AddNode("B", providesInt)
	assert.NoError(t, err)
	_, err = r.Resolve()
	assert.NoError(t, err, "valid dep chain caused error")

	r = NewResolver()
	err = r.AddNode("A", returnsNilErr)
	assert.NoError(t, err)
	_, err = r.Resolve()
	assert.NoError(t, err, "returning nil error caused error")

	r = NewResolver()
	err = r.AddNode("A", returnsNonNilError)
	assert.NoError(t, err)
	_, err = r.Resolve()
	assert.Error(t, err, "returning non-nil error did not cause error")

	r = NewResolver()
	err = r.AddNode("1", cyclePartOne)
	assert.NoError(t, err)
	err = r.AddNode("2", cyclePartTwo)
	assert.NoError(t, err)
	_, err = r.Resolve()
	assert.Error(t, err, "cycle did not cause error")

	// Note that this shouldn't be possible to hit
	r = NewResolver()
	n, err := newFuncNode("A", needsInt)
	assert.NoError(t, err)
	n.requires = nil // this will mess up the internal topo sort so we can get to the inject error
	r.nodes = append(r.nodes, n)
	_, err = r.Resolve()
	assert.Error(t, err, "injector did not cause error for requiring missing types")
}
