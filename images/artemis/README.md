# artemis

This module provides Apache Artemis support (docker container) for [marrow](https://github.com/go-andiamo/marrow)

## Installation

    go get github.com/go-andiamo/marrow/images/artemis

## Usage

Example of how to use this image support for [marrow](https://github.com/go-andiamo/marrow):

```go
package main

import (
    "github.com/go-andiamo/marrow"
    "github.com/go-andiamo/marrow/images/artemis"
)

func main() {
    s := marrow.Suite(endpoints...).Init(
        artemis.With(artemis.Options{}),
    )
    err := s.Run()
    if err != nil {
        panic(err)
    }
}
```
