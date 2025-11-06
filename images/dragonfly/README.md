# dragonfly

Dragonfly is essentially a drop-in replacement for Redis

This module provides Dragonfly support (docker container) for [marrow](https://github.com/go-andiamo/marrow)

## Installation

    go get github.com/go-andiamo/marrow/images/dragonfly

## Usage

Example of how to use this image support for [marrow](https://github.com/go-andiamo/marrow):

```go
package main

import (
    "github.com/go-andiamo/marrow"
    "github.com/go-andiamo/marrow/images/dragonfly"
)

func main() {
    s := marrow.Suite(endpoints...).Init(
        dragonfly.With(dragonfly.Options{}),
    )
    err := s.Run()
    if err != nil {
        panic(err)
    }
}
```
