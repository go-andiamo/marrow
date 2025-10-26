package coverage

import (
	"github.com/go-andiamo/marrow/common"
	"io"
	"net/http"
	"time"
)

type Collector interface {
	LoadSpec(r io.Reader) (err error)
	ReportFailure(endpoint common.Endpoint, method common.Method, req *http.Request, err error)
	ReportUnmet(endpoint common.Endpoint, method common.Method, req *http.Request, exp common.Expectation, err error)
	ReportMet(endpoint common.Endpoint, method common.Method, req *http.Request, exp common.Expectation)
	ReportSkipped(endpoint common.Endpoint, method common.Method, req *http.Request, exp common.Expectation)
	ReportTiming(endpoint common.Endpoint, method common.Method, req *http.Request, dur time.Duration)
	HasFailures() bool
}
