package coverage

import (
	"context"
	"github.com/go-andiamo/chioas"
	"github.com/go-andiamo/marrow/common"
	"io"
	"net/http"
	"sync"
	"time"
)

func NewCoverage() *Coverage {
	return &Coverage{
		Endpoints:       make(map[string]*Endpoint),
		normalizedPaths: make(map[string]map[string]struct{}),
	}
}

// Coverage is the default coverage information
type Coverage struct {
	Endpoints map[string]*Endpoint
	OAS       *chioas.Definition
	Common
	mutex           sync.RWMutex
	normalizedPaths map[string]map[string]struct{}
}

var _ Collector = (*Coverage)(nil)

func (c *Coverage) ReportFailure(endpoint common.Endpoint, method common.Method, req *http.Request, err error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	fail := Failure{
		Endpoint: endpoint,
		Method:   method,
		Request:  requestShallowClone(req),
		Error:    err,
	}
	covE, covM := c.add(endpoint, method)
	if covE != nil {
		covE.Failures = append(covE.Failures, fail)
	}
	if covM != nil {
		covM.Failures = append(covM.Failures, fail)
	}
	c.Failures = append(c.Failures, fail)
}

func (c *Coverage) ReportUnmet(endpoint common.Endpoint, method common.Method, req *http.Request, exp common.Expectation, err error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	unmet := Unmet{
		Endpoint:    endpoint,
		Method:      method,
		Request:     requestShallowClone(req),
		Expectation: exp,
		Error:       err,
	}
	covE, covM := c.add(endpoint, method)
	if covE != nil {
		covE.Unmet = append(covE.Unmet, unmet)
	}
	if covM != nil {
		covM.Unmet = append(covM.Unmet, unmet)
	}
	c.Unmet = append(c.Unmet, unmet)
}

func (c *Coverage) ReportMet(endpoint common.Endpoint, method common.Method, req *http.Request, exp common.Expectation) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	met := Met{
		Endpoint:    endpoint,
		Method:      method,
		Request:     requestShallowClone(req),
		Expectation: exp,
	}
	covE, covM := c.add(endpoint, method)
	if covE != nil {
		covE.Met = append(covE.Met, met)
	}
	if covM != nil {
		covM.Met = append(covM.Met, met)
	}
	c.Met = append(c.Met, met)
}

func (c *Coverage) ReportSkipped(endpoint common.Endpoint, method common.Method, req *http.Request, exp common.Expectation) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	skip := Skip{
		Endpoint:    endpoint,
		Method:      method,
		Request:     requestShallowClone(req),
		Expectation: exp,
	}
	covE, covM := c.add(endpoint, method)
	if covE != nil {
		covE.Skipped = append(covE.Skipped, skip)
	}
	if covM != nil {
		covM.Skipped = append(covM.Skipped, skip)
	}
	c.Skipped = append(c.Skipped, skip)
}

func (c *Coverage) ReportTiming(endpoint common.Endpoint, method common.Method, req *http.Request, dur time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	timing := Timing{
		Endpoint: endpoint,
		Method:   method,
		Request:  requestShallowClone(req),
		Duration: dur,
	}
	covE, covM := c.add(endpoint, method)
	if covE != nil {
		covE.Timings = append(covE.Timings, timing)
	}
	if covM != nil {
		covM.Timings = append(covM.Timings, timing)
	}
	c.Timings = append(c.Timings, timing)
}

func (c *Coverage) HasFailures() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return len(c.Failures) > 0 || len(c.Unmet) > 0
}

func requestShallowClone(req *http.Request) *http.Request {
	if req == nil {
		return nil
	}
	// Clone copies URL, Header, etc., with a fresh context.
	r2 := req.Clone(context.Background())
	// nuke live/streaming fields so we never hold sockets/buffers...
	r2.Body = http.NoBody
	r2.GetBody = func() (io.ReadCloser, error) {
		return http.NoBody, nil
	}
	r2.Trailer = nil
	r2.MultipartForm = nil
	return r2
}

func (c *Coverage) addNormalizedPath(endpoint common.Endpoint) {
	path := endpoint.Url()
	nPath := normalizePath(path)
	if m, ok := c.normalizedPaths[nPath]; ok {
		m[path] = struct{}{}
	} else {
		c.normalizedPaths[nPath] = map[string]struct{}{path: {}}
	}
}

func (c *Coverage) add(endpoint common.Endpoint, method common.Method) (covE *Endpoint, covM *Method) {
	if endpoint != nil {
		c.addNormalizedPath(endpoint)
		var ok bool
		if covE, ok = c.Endpoints[endpoint.Url()]; !ok {
			covE = &Endpoint{
				Endpoint: endpoint,
				Methods:  make(map[string]*Method),
			}
			c.Endpoints[endpoint.Url()] = covE
		}
		if method != nil {
			if covM, ok = covE.Methods[method.MethodName()]; !ok {
				covM = &Method{
					Method: method,
				}
				covE.Methods[method.MethodName()] = covM
			}
		}
	}
	return covE, covM
}
