# redis7



This module provides Redis support (docker container) for [marrow](https://github.com/go-andiamo/marrow)

## Installation

    go get github.com/go-andiamo/marrow/images/redis7

## Usage

Example of how to use this image support for [marrow](https://github.com/go-andiamo/marrow):

```go
package main

import (
    "github.com/go-andiamo/marrow"
    "module github.com/go-andiamo/marrow/images/redis7"
)

func main() {
    s := marrow.Suite(endpoints...).Init(
        redis7.With(redis7.Options{}),
    )
    err := s.Run()
    if err != nil {
        panic(err)
    }
}
```
