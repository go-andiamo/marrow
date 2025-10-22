package marrow

func newCoverage() *Coverage {
	return &Coverage{
		Endpoints: map[string]*CoverageEndpoint{},
	}
}

type Coverage struct {
	Endpoints map[string]*CoverageEndpoint
	Failures  []CoverageFailure
}

func (c *Coverage) reportFailure(endpoint Endpoint_, method Method_, err error) {
	if covEnpoint, covMethod := c.add(endpoint, method); covMethod != nil {
		covMethod.Failures = append(covMethod.Failures, CoverageFailure{
			Endpoint: endpoint,
			Method:   method,
			Error:    err,
		})
	} else if covEnpoint != nil {
		covEnpoint.Failures = append(covEnpoint.Failures, CoverageFailure{
			Endpoint: endpoint,
			Error:    err,
		})
	} else {
		c.Failures = append(c.Failures, CoverageFailure{
			Error: err,
		})
	}
}

func (c *Coverage) reportUnmet(endpoint Endpoint_, method Method_, exp Expectation, err error) {
	if _, covMethod := c.add(endpoint, method); covMethod != nil {
		covMethod.Unmet = append(covMethod.Unmet, CoverageUnmet{
			Endpoint:    endpoint,
			Method:      method,
			Expectation: exp,
			Error:       err,
		})
	}
}

func (c *Coverage) reportMet(endpoint Endpoint_, method Method_, exp Expectation) {
	if _, covMethod := c.add(endpoint, method); covMethod != nil {
		covMethod.Met = append(covMethod.Met, CoverageMet{
			Endpoint:    endpoint,
			Method:      method,
			Expectation: exp,
		})
	}
}

func (c *Coverage) add(endpoint Endpoint_, method Method_) (covPath *CoverageEndpoint, covMethod *CoverageMethod) {
	if endpoint != nil {
		var ok bool
		if covPath, ok = c.Endpoints[endpoint.Url()]; !ok {
			covPath = &CoverageEndpoint{
				Endpoint: endpoint,
				Methods:  make(map[string]*CoverageMethod),
			}
			c.Endpoints[endpoint.Url()] = covPath
		}
		if method != nil {
			if covMethod, ok = covPath.Methods[string(method.Method())]; !ok {
				covMethod = &CoverageMethod{
					Method: method,
				}
				covPath.Methods[string(method.Method())] = covMethod
			}
		}
	}
	return nil, nil
}

type CoverageFailure struct {
	Endpoint Endpoint_
	Method   Method_
	Error    error
}

type CoverageEndpoint struct {
	Endpoint Endpoint_
	Failures []CoverageFailure
	Methods  map[string]*CoverageMethod
}

type CoverageMethod struct {
	Method   Method_
	Failures []CoverageFailure
	Unmet    []CoverageUnmet
	Met      []CoverageMet
}

type CoverageMet struct {
	Endpoint    Endpoint_
	Method      Method_
	Expectation Expectation
}

type CoverageUnmet struct {
	Endpoint    Endpoint_
	Method      Method_
	Expectation Expectation
	Error       error
}
