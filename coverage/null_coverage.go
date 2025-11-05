package coverage

import (
	"github.com/go-andiamo/marrow/common"
	"io"
	"net/http"
	"time"
)

func NewNullCoverage() Collector {
	return &nullCoverage{}
}

type nullCoverage struct {
	hasFailures bool
}

var _ Collector = (*nullCoverage)(nil)

func (n *nullCoverage) LoadSpec(r io.Reader) (err error) {
	// nullCoverage does nothing
	return nil
}

func (n *nullCoverage) ReportFailure(endpoint common.Endpoint, method common.Method, req *http.Request, err error) {
	n.hasFailures = true
	// nullCoverage does nothing
}

func (n *nullCoverage) ReportUnmet(endpoint common.Endpoint, method common.Method, req *http.Request, exp common.Expectation, err error) {
	n.hasFailures = true
	// nullCoverage does nothing
}

func (n *nullCoverage) ReportMet(endpoint common.Endpoint, method common.Method, req *http.Request, exp common.Expectation) {
	// nullCoverage does nothing
}

func (n *nullCoverage) ReportSkipped(endpoint common.Endpoint, method common.Method, req *http.Request, exp common.Expectation) {
	// nullCoverage does nothing
}

func (n *nullCoverage) ReportTiming(endpoint common.Endpoint, method common.Method, req *http.Request, dur time.Duration, tt *TraceTiming) {
	// nullCoverage does nothing
}

func (n *nullCoverage) HasFailures() bool {
	return n.hasFailures
}
