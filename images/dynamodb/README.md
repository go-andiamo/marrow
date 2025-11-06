# dynamodb

This module provides DynamoDB support (docker container - localstack) for [marrow](https://github.com/go-andiamo/marrow)

## Installation

    go get github.com/go-andiamo/marrow/images/dynamodb

## Usage

Example of how to use this image support for [marrow](https://github.com/go-andiamo/marrow):

```go
package main

import (
    "github.com/go-andiamo/marrow"
    "github.com/go-andiamo/marrow/images/dynamodb"
)

func main() {
    s := marrow.Suite(endpoints...).Init(
        dynamodb.With("dynamodb", dynamodb.Options{}),
    )
    err := s.Run()
    if err != nil {
        panic(err)
    }
}
```
