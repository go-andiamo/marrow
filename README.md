# Marrow
[![GoDoc](https://godoc.org/github.com/go-andiamo/marrow?status.svg)](https://pkg.go.dev/github.com/go-andiamo/marrow)
[![Latest Version](https://img.shields.io/github/v/tag/go-andiamo/marrow.svg?sort=semver&style=flat&label=version&color=blue)](https://github.com/go-andiamo/marrow/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/go-andiamo/marrow)](https://goreportcard.com/report/github.com/go-andiamo/marrow)

An API integration test framework - for testing APIs written in Go, using a framework written in Go, with tests written in Go.

## Features

- **Composable DSL for end-to-end API scenarios**  
  Describe tests as readable, fluent flows – no test YAML, no magic, just Go.

- **Black-box testing of any API**  
  Hit a real HTTP endpoint and assert on both the response *and* what it did to its dependencies. Works regardless of what language/framework the API is written in.

- **Real dependency “wraps” (not mocks)**  
  Spin up real services as test dependencies (e.g. MongoDB, LocalStack/AWS services, message brokers) and assert on their state and events.

- **Full coverage support**  
  Execute the API tests and optionally report endpoint coverage (even against a provided OAS spec)  
  Coverage reports can also supply timings (inc. averages, variance, P50, P90, P99) - with built-in support for repeated runs.

- **Resolvable values**  
  Uniform mechanism for “values that come from somewhere”: JSON fields, variables, database queries, message payloads, lengths, first/last elements, etc.

- **Powerful assertions**  
  Rich set of assertion helpers over response payloads, headers, status codes, dependency state, and captured events.

- **Variables**  
  Declare vars once, `SetVar()` from responses or dependencies, and reuse them across steps. IDE refactoring-friendly because var names are just Go identifiers.

- **Conditional flows**  
  `If(...)` / `IfNot(...)` blocks let you express conditional logic directly in the DSL without branching mess in test code.

- **Capturing and inspecting side effects**  
  Capture SNS/SQS-style publishes, queue messages, Mongo changes, etc., and assert on count, content, and ordering.

- **Composable helpers**  
  Small building blocks (values, assertions, conditionals, captures) that can be combined arbitrarily – everything is designed to be reused and extended.

- **Extensible dependency model**  
  Built-in examples (Mongo, LocalStack, etc.) double as templates for adding your own dependency “wraps” with minimal boilerplate.

- **First-class Go testing integration**  
  Plays nicely with `testing.T`, subtests, and your existing tooling; tests are just Go code, no additional runner or framework ceremony.

- **Build freshness guard (`make` integration)**  
  Optionally run a `make` target before tests, so every scenario runs against the latest build artefact instead of whatever binary happened to be lying around.

- **IDE-friendly & CI/CD-friendly**  
  Runs identically in GoLand, VS Code, your terminal, GitHub Actions, GitLab CI, Jenkins, or any CI/CD pipeline.  
  No custom runners, no plugins, no hidden runtime - just `go test`.


## Design Philosophy

The intent of the design is to describe API tests in a non-abstract DSL (Domain Specific Language) that can be used by developers and QAs alike.

We've specifically avoided terms like "scenario" - instead using terms like "endpoint" & "method" to describe what's being tested.

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

And to run the endpoints tests...

```go
package main

import (
    . "github.com/go-andiamo/marrow"
)

func main() {
    s := Suite(endpoints...)
    err := s.Init(/* whatever initializers needed */).Run()
    if err != nil {
        panic(err)
    }
}
```

Or to run as part of Go tests...

```go
package main

import (
    . "github.com/go-andiamo/marrow"
    "github.com/go-andiamo/marrow/with"
    "testing"
)

func TestApi(t *testing.T) {
    s := Suite(endpoints...)
    err := s.Init(
        with.Testing(t),
        /* whatever other initializers needed */).Run()
    if err != nil {
        t.Fatal(err)
    }
}
```

## Installation

    go get github.com/go-andiamo/marrow

## Supporting images

_Marrow_ comes with several ready-rolled supporting images for common dependencies (more to come)...

- [Dragonfly](https://github.com/go-andiamo/marrow/tree/main/images/dragonfly) (drop-in replacement for Redis)  
  `go get github.com/go-andiamo/marrow/images/dragonfly`
- [Kafka](https://github.com/go-andiamo/marrow/tree/main/images/kafka)  
  `go get github.com/go-andiamo/marrow/images/kafka`
- [AWS localstack](https://github.com/go-andiamo/marrow/tree/main/images/localstack)  
  `go get github.com/go-andiamo/marrow/images/localstack`
- [MongoDB](https://github.com/go-andiamo/marrow/tree/main/images/mongo)  
  `go get github.com/go-andiamo/marrow/images/mongo`
- [MySql](https://github.com/go-andiamo/marrow/tree/main/images/mysql)  
  `go get github.com/go-andiamo/marrow/images/mysql`
- [Postgres](https://github.com/go-andiamo/marrow/tree/main/images/postgres)  
  `go get github.com/go-andiamo/marrow/images/postgres`
- [Redis](https://github.com/go-andiamo/marrow/tree/main/images/redis7)  
  `go get github.com/go-andiamo/marrow/images/redis7`

Support images can also be used independently of _Marrow_ in unit testing - see [example](https://github.com/go-andiamo/marrow/blob/main/_examples/petstore_db_image/repository/repository_test.go)