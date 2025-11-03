# Marrow

An API integration test framework — for testing APIs written in Go, using a framework written in Go, with tests written in Go.

## Design Philosophy

The intent of the design is to describe API tests in a non-abstract DSL (Domain Specific Language) that can be used by developers and QAs alike.

We've specifically avoided terms like "scenario" - instead using terms like "endpoint" & "method" to describe what's being tested.

Having provided a description of the endpoints and methods to be tested (and various asserts/requires) - the test suite can be run either as a Golang test or its own test runner.

Comprehensive support for spinning up dependencies (as Docker containers - e.g. databases) to be used by the API being tested.