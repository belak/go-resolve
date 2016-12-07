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

## Other stuff

Be sure to check out [go-plugin](https://github.com/belak/go-plugin),
which was my main reason for writing this.
