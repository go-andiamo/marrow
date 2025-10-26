package coverage

import (
	"github.com/go-andiamo/marrow/common"
	"net/http"
)

type Endpoint struct {
	Endpoint common.Endpoint
	Methods  map[string]*Method
	Common
}

type Method struct {
	Method common.Method
	Common
}

type Common struct {
	Failures []Failure
	Unmet    []Unmet
	Met      []Met
	Skipped  []Skip
	Timings  Timings
}

type Failure struct {
	Endpoint common.Endpoint
	Method   common.Method
	Request  *http.Request
	Error    error
}

type Unmet struct {
	Endpoint    common.Endpoint
	Method      common.Method
	Request     *http.Request
	Expectation common.Expectation
	Error       error
}

type Met struct {
	Endpoint    common.Endpoint
	Method      common.Method
	Request     *http.Request
	Expectation common.Expectation
}

type Skip struct {
	Endpoint    common.Endpoint
	Method      common.Method
	Request     *http.Request
	Expectation common.Expectation
}
