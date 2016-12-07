# go-resolve

[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg)](https://godoc.org/github.com/belak/go-resolve)
[![Travis](https://img.shields.io/travis/belak/go-resolve.svg)](https://travis-ci.org/belak/go-resolve)
[![Coveralls](https://img.shields.io/coveralls/belak/go-resolve.svg)](https://coveralls.io/github/belak/go-resolve)

## Why write this? Isn't dependency injection bad?

After messing with plugin systems for a while, one of the hardest
problems to solve is plugin dependencies and loading things in a
certain order. For generic systems, it can be hard to handle this
properly in a safe manner.

`go-resolve` is intended to be a small wrapper around dependency
injection to determine the proper load order of factory-like plugins
and load them.

Dependency injection is generally frowned upon in go, but I view this
as an acceptable trade-off to make writing plugin systems
simpler. Additionally, running dependency injection once on startup,
rather than every time an event happens, makes this simpler to manage
and reduces the problem surface to the point where it should be
testable.

## But empty interfaces are bad!

Yes, this is true. Using empty interfaces in too many places is
generally a code smell. However, in order to accept a function with
any signature, this is a requirement.

We work around this by checking that the function will be valid for
injection at the time it's added.

## How does it work?

The actual concept behind ordering dependencies is fairly
simple. Essentially, a dependency graph needs to be created, then you
can just run a topological sort on the graph. The order of the nodes
from that sort is the order items needs to be loaded in.

When we add a function to a resolver, we generate a graph node which
contains a list of all the types needed to run that function as well
as which types are returned. Any error types are ignored until the
function itself is run.

After all nodes are added, simply call Resolve. This will determine
the load order, load all plugins, and add any non-error return values
to an Injector for use after plugins are loaded. If any function
returns a non-nil error value, that will be returned as soon as it
happens.

## Other stuff

Be sure to check out [go-plugin](https://github.com/belak/go-plugin),
which was my main reason for writing this.
