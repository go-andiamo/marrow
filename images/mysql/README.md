# mysql

This module provides MySql support (docker container - localstack) for [marrow](https://github.com/go-andiamo/marrow)

## Installation

    go get github.com/go-andiamo/marrow/images/mysql

## Usage

Example of how to use this image support for [marrow](https://github.com/go-andiamo/marrow):

```go
package main

import (
    "github.com/go-andiamo/marrow"
    "github.com/go-andiamo/marrow/images/mysql"
)

func main() {
    s := marrow.Suite(endpoints...).Init(
        mysql.With("mysql", mysql.Options{}),
    )
    err := s.Run()
    if err != nil {
        panic(err)
    }
}
```
