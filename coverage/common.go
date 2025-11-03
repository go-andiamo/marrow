package coverage

import (
	"github.com/go-andiamo/marrow/common"
	"net/http"
)

// Endpoint provides coverage information about an endpoint
type Endpoint struct {
	Endpoint common.Endpoint
	Methods  map[string]*Method
	Common
}

// Method provides coverage information about a method
type Method struct {
	Method common.Method
	Common
}

// Common provides common coverage information
type Common struct {
	Failures []Failure
	Unmet    []Unmet
	Met      []Met
	Skipped  []Skip
	Timings  Timings
}

// Failure provides coverage information about a failure
type Failure struct {
	Endpoint common.Endpoint
	Method   common.Method
	Request  *http.Request
	Error    error
}

// Unmet provides coverage information about an unmet expectation
type Unmet struct {
	Endpoint    common.Endpoint
	Method      common.Method
	Request     *http.Request
	Expectation common.Expectation
	Error       error
}

// Met provides coverage information about a met expectation
type Met struct {
	Endpoint    common.Endpoint
	Method      common.Method
	Request     *http.Request
	Expectation common.Expectation
}

// Skip provides coverage information about a skipped expectation
type Skip struct {
	Endpoint    common.Endpoint
	Method      common.Method
	Request     *http.Request
	Expectation common.Expectation
}
