# nats

This module provides Nats support for [marrow](https://github.com/go-andiamo/marrow)

## Installation

    go get github.com/go-andiamo/marrow/images/nats

## Usage

Example of how to use this image support for [marrow](https://github.com/go-andiamo/marrow):

```go
package main

import (
    "github.com/go-andiamo/marrow"
    "github.com/go-andiamo/marrow/images/nats"
)

func main() {
    s := marrow.Suite(endpoints...).Init(
        nats.With(nats.Options{}),
    )
    err := s.Run()
    if err != nil {
        panic(err)
    }
}
```
