package marrow

func newCoverage() *Coverage {
	return &Coverage{
		Endpoints: map[string]*CoverageEndpoint{},
	}
}

type Coverage struct {
	Endpoints map[string]*CoverageEndpoint
	Failures  []CoverageFailure
	Unmet     []CoverageUnmet
	Met       []CoverageMet
}

func (c *Coverage) reportFailure(endpoint Endpoint_, method Method_, err error) {
	fail := CoverageFailure{
		Endpoint: endpoint,
		Method:   method,
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

func (c *Coverage) reportUnmet(endpoint Endpoint_, method Method_, exp Expectation, err error) {
	unmet := CoverageUnmet{
		Endpoint:    endpoint,
		Method:      method,
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

func (c *Coverage) reportMet(endpoint Endpoint_, method Method_, exp Expectation) {
	met := CoverageMet{
		Endpoint:    endpoint,
		Method:      method,
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

func (c *Coverage) add(endpoint Endpoint_, method Method_) (covE *CoverageEndpoint, covM *CoverageMethod) {
	if endpoint != nil {
		var ok bool
		if covE, ok = c.Endpoints[endpoint.Url()]; !ok {
			covE = &CoverageEndpoint{
				Endpoint: endpoint,
				Methods:  make(map[string]*CoverageMethod),
			}
			c.Endpoints[endpoint.Url()] = covE
		}
		if method != nil {
			if covM, ok = covE.Methods[string(method.Method())]; !ok {
				covM = &CoverageMethod{
					Method: method,
				}
				covE.Methods[string(method.Method())] = covM
			}
		}
	}
	return covE, covM
}

type CoverageFailure struct {
	Endpoint Endpoint_
	Method   Method_
	Error    error
}

type CoverageEndpoint struct {
	Endpoint Endpoint_
	Failures []CoverageFailure
	Unmet    []CoverageUnmet
	Met      []CoverageMet
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
