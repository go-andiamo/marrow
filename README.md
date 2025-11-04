# Marrow
[![GoDoc](https://godoc.org/github.com/go-andiamo/marrow?status.svg)](https://pkg.go.dev/github.com/go-andiamo/marrow/typed)
[![Latest Version](https://img.shields.io/github/v/tag/go-andiamo/marrow.svg?sort=semver&style=flat&label=version&color=blue)](https://github.com/go-andiamo/marrow/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/go-andiamo/marrow)](https://goreportcard.com/report/github.com/go-andiamo/marrow)

An API integration test framework — for testing APIs written in Go, using a framework written in Go, with tests written in Go.

## Design Philosophy

The intent of the design is to describe API tests in a non-abstract DSL (Domain Specific Language) that can be used by developers and QAs alike.

We've specifically avoided terms like "scenario" - instead using terms like "endpoint" & "method" to describe what's being tested.

Having provided a description of the endpoints and methods to be tested (and various asserts/requires) - the test suite can be run either as a Golang test or its own test runner.

Comprehensive support for spinning up dependencies (as Docker containers - e.g. databases) to be used by the API being tested.

## Example

Full working demo examples can be found in [Petstore](https://github.com/go-andiamo/marrow/tree/main/_examples/petstore/_integration_tests) and [Petstore with db and API image](https://github.com/go-andiamo/marrow/tree/main/_examples/petstore_db_image/_integration_tests)

An example of what the tests definition looks like...

```go
package main

import (
    . "github.com/go-andiamo/marrow"
)

var endpointTests = []Endpoint_{
    Endpoint("/api", "Root",
        Method(GET, "Get root").
            AssertOK().
            AssertEqual(JsonPath(Body, "hello"), "world"),
        Endpoint("/categories", "Categories",
            Method(GET, "Get first category id (used for creating pet)").
                RequireOK().
                RequireGreaterThan(JsonPath(Body, LEN), 0).
                SetVar(After, "category-id", JsonPath(JsonPath(Body, "0"), "id")),
        ),
        Endpoint("/pets", "Pets",
            Method(GET, "Get pets (empty)").
                AssertOK().
                AssertLen(Body, 0),
            Method(POST, "Create pet").
                RequestBody(JSON{
                    "name": "Felix",
                    "dob":  "2025-11-01",
                    "category": JSON{
                        "id": Var("category-id"),
                    },
                }).
                AssertCreated().
                SetVar(After, "created-pet-id", JsonPath(Body, "id")),
            Endpoint("/{petId}", "Pet",
                Method(GET, "Get pet (not found)").
                    PathParam(Var("non-uuid")).
                    AssertNotFound(),
                Method(PUT, "Update pet (not found)").
                    PathParam(Var("non-uuid")).
                    AssertNotFound(),
                Method(PUT, "Update pet successful").
                    PathParam(Var("created-pet-id")).
                    RequestBody(JSON{
                        "name": "Feline",
                        "dob":  "2025-11-02",
                        "category": JSON{
                            "id": Var("category-id"),
                        },
                    }).
                    AssertOK(),
                Method(DELETE, "Delete pet (not found)").
                    PathParam(Var("non-uuid")).
                    AssertNotFound(),
                Method(DELETE, "Delete pet successful").
                    PathParam(Var("created-pet-id")).
                    AssertNoContent(),
            ),
        ),
        Endpoint("/categories", "Categories",
            Method(GET, "Get categories").
                AssertOK().
                AssertGreaterThan(JsonPath(Body, LEN), 0),
            Endpoint("/{categoryId}", "Category",
                Method(GET, "Get category (not found)").
                    PathParam(Var("non-uuid")).
                    AssertNotFound(),
                Method(GET, "Get category (found)").
                    PathParam(Var("category-id")).
                    AssertOK(),
            ),
        ),
    ),
}
```