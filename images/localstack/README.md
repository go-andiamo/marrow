# localstack

This module provides AWS localstack support (docker container) for [marrow](https://github.com/go-andiamo/marrow)

## Installation

    go get github.com/go-andiamo/marrow/images/localstack

## Usage

Example of how to use this image support for [marrow](https://github.com/go-andiamo/marrow):

```go
package main

import (
    "github.com/go-andiamo/marrow"
    "github.com/go-andiamo/marrow/images/localstack"
)

func main() {
    s := marrow.Suite(endpoints...).Init(
        localstack.With(localstack.Options{Sevices: localstack.Services{localstack.All}}),
    )
    err := s.Run()
    if err != nil {
        panic(err)
    }
}
```
