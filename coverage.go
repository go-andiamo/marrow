package marrow

import (
	"bufio"
	ctx "context"

	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-andiamo/chioas"
	"github.com/go-andiamo/splitter"
	"gopkg.in/yaml.v3"
	"io"
	"math"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

func newCoverage() *Coverage {
	return &Coverage{
		Endpoints:       make(map[string]*CoverageEndpoint),
		normalizedPaths: make(map[string]map[string]struct{}),
	}
}

type CoverageCollector interface {
	LoadSpec(r io.Reader) (err error)
	ReportFailure(endpoint Endpoint_, method Method_, req *http.Request, err error)
	ReportUnmet(endpoint Endpoint_, method Method_, req *http.Request, exp Expectation, err error)
	ReportMet(endpoint Endpoint_, method Method_, req *http.Request, exp Expectation)
	ReportSkipped(endpoint Endpoint_, method Method_, req *http.Request, exp Expectation)
	ReportTiming(endpoint Endpoint_, method Method_, req *http.Request, dur time.Duration)
}

type CoverageCommon struct {
	Failures []CoverageFailure
	Unmet    []CoverageUnmet
	Met      []CoverageMet
	Skipped  []CoverageSkip
	Timings  CoverageTimings
}

type Coverage struct {
	Endpoints map[string]*CoverageEndpoint
	OAS       *chioas.Definition
	CoverageCommon
	mutex           sync.Mutex
	normalizedPaths map[string]map[string]struct{}
}

var _ CoverageCollector = (*Coverage)(nil)

func (c *Coverage) LoadSpec(r io.Reader) (err error) {
	br := bufio.NewReader(r)
	var spec *chioas.Definition
	var first []byte
	// sniff for json or yaml...
	if first, err = br.Peek(1); err == nil {
		spec = new(chioas.Definition)
		if first[0] == '{' {
			err = json.NewDecoder(br).Decode(spec)
		} else {
			err = yaml.NewDecoder(br).Decode(spec)
		}
	}
	if err != nil {
		return fmt.Errorf("unable to read OAS: %w", err)
	} else {
		c.OAS = spec
		return nil
	}
}

type CoverageEndpoint struct {
	Endpoint Endpoint_
	Methods  map[string]*CoverageMethod
	CoverageCommon
}

type CoverageMethod struct {
	Method Method_
	CoverageCommon
}

func (c *Coverage) ReportFailure(endpoint Endpoint_, method Method_, req *http.Request, err error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	fail := CoverageFailure{
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

func (c *Coverage) ReportUnmet(endpoint Endpoint_, method Method_, req *http.Request, exp Expectation, err error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	unmet := CoverageUnmet{
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

func (c *Coverage) ReportMet(endpoint Endpoint_, method Method_, req *http.Request, exp Expectation) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	met := CoverageMet{
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

func (c *Coverage) ReportSkipped(endpoint Endpoint_, method Method_, req *http.Request, exp Expectation) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	skip := CoverageSkip{
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

func (c *Coverage) ReportTiming(endpoint Endpoint_, method Method_, req *http.Request, dur time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	timing := CoverageTiming{
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

func requestShallowClone(req *http.Request) *http.Request {
	// Clone copies URL, Header, etc., with a fresh context.
	r2 := req.Clone(ctx.Background())
	// nuke live/streaming fields so we never hold sockets/buffers...
	r2.Body = http.NoBody
	r2.GetBody = nil
	r2.Trailer = nil
	r2.MultipartForm = nil
	return r2
}

var normSplitter = splitter.MustCreateSplitter('/', splitter.CurlyBrackets).AddDefaultOptions(
	splitter.IgnoreEmptyFirst,
	splitter.IgnoreEmptyLast,
	&normCapture{})

type normCapture struct{}

func (nc *normCapture) Apply(s string, pos int, totalLen int, captured int, skipped int, isLast bool, subParts ...splitter.SubPart) (cap string, add bool, err error) {
	add = true
	cap = s
	if strings.HasPrefix(cap, "{") && strings.HasSuffix(cap, "}") {
		cap = "{}"
	}
	return
}

func normalizePath(path string) string {
	if parts, err := normSplitter.Split(path); err == nil {
		return "/" + strings.Join(parts, "/")
	} else {
		return path
	}
}

func (c *Coverage) addNormalizedPath(endpoint Endpoint_) {
	path := endpoint.Url()
	nPath := normalizePath(path)
	if m, ok := c.normalizedPaths[nPath]; ok {
		m[path] = struct{}{}
	} else {
		c.normalizedPaths[nPath] = map[string]struct{}{path: {}}
	}
}

func (c *Coverage) add(endpoint Endpoint_, method Method_) (covE *CoverageEndpoint, covM *CoverageMethod) {
	if endpoint != nil {
		c.addNormalizedPath(endpoint)
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
	Request  *http.Request
	Error    error
}

type CoverageUnmet struct {
	Endpoint    Endpoint_
	Method      Method_
	Request     *http.Request
	Expectation Expectation
	Error       error
}

type CoverageMet struct {
	Endpoint    Endpoint_
	Method      Method_
	Request     *http.Request
	Expectation Expectation
}

type CoverageSkip struct {
	Endpoint    Endpoint_
	Method      Method_
	Request     *http.Request
	Expectation Expectation
}

type SpecCoverage struct {
	CoveredPaths    map[string]*SpecPathCoverage
	NonCoveredPaths map[string]*SpecPathCoverage
	UnknownPaths    map[string]*SpecPathCoverage
}

func (s *SpecCoverage) PathsCovered() (total int, covered int, perc float64) {
	covered = len(s.CoveredPaths)
	total = covered + len(s.NonCoveredPaths)
	perc = float64(covered) / float64(total)
	return
}

func (s *SpecCoverage) MethodsCovered() (total int, covered int, perc float64) {
	for _, cp := range s.CoveredPaths {
		total += len(cp.CoveredMethods) + len(cp.NonCoveredMethods)
		covered += len(cp.CoveredMethods)
	}
	for _, cp := range s.NonCoveredPaths {
		total += len(cp.PathDef.Methods)
	}
	perc = float64(covered) / float64(total)
	return
}

func newSpecCoverage() *SpecCoverage {
	return &SpecCoverage{
		CoveredPaths:    make(map[string]*SpecPathCoverage),
		NonCoveredPaths: make(map[string]*SpecPathCoverage),
		UnknownPaths:    make(map[string]*SpecPathCoverage),
	}
}

type SpecPathCoverage struct {
	Path              string
	PathDef           *chioas.Path
	CoveredMethods    map[string]*SpecMethodCoverage
	NonCoveredMethods map[string]*SpecMethodCoverage
	UnknownMethods    map[string]*SpecMethodCoverage
}

func newSpecPathCoverage(path string, def *chioas.Path) *SpecPathCoverage {
	return &SpecPathCoverage{
		Path:              path,
		PathDef:           def,
		CoveredMethods:    make(map[string]*SpecMethodCoverage),
		NonCoveredMethods: make(map[string]*SpecMethodCoverage),
		UnknownMethods:    make(map[string]*SpecMethodCoverage),
	}
}

type SpecMethodCoverage struct {
	Method    string
	MethodDef *chioas.Method
	CoverageCommon
}

func (c *Coverage) checkMethodCoverage(pathCov *SpecPathCoverage, methods chioas.Methods, tPaths map[string]struct{}) {
	for m, mDef := range methods {
		mCov := &SpecMethodCoverage{
			Method:    m,
			MethodDef: &mDef,
		}
		found := false
		for tPath := range tPaths {
			if cc, ok := c.Endpoints[tPath]; ok {
				if cm, ok := cc.Methods[m]; ok {
					found = true
					mCov.Met = append(mCov.Met, cm.Met...)
					mCov.Unmet = append(mCov.Unmet, cm.Unmet...)
					mCov.Failures = append(mCov.Failures, cm.Failures...)
					mCov.Skipped = append(mCov.Skipped, cm.Skipped...)
					mCov.Timings = append(mCov.Timings, cm.Timings...)
				}
			}
		}
		if found {
			pathCov.CoveredMethods[m] = mCov
		} else {
			pathCov.NonCoveredMethods[m] = mCov
		}
	}
	for tPath := range tPaths {
		if cc, ok := c.Endpoints[tPath]; ok {
			for m, cm := range cc.Methods {
				if _, ok := methods[m]; !ok {
					if um, ok := pathCov.UnknownMethods[m]; ok {
						um.Failures = append(um.Failures, cm.Failures...)
						um.Unmet = append(um.Unmet, cm.Unmet...)
						um.Met = append(um.Met, cm.Met...)
						um.Skipped = append(um.Skipped, cm.Skipped...)
						um.Timings = append(um.Timings, cm.Timings...)
					} else {
						pathCov.UnknownMethods[m] = &SpecMethodCoverage{
							Method: m,
							CoverageCommon: CoverageCommon{
								Failures: append([]CoverageFailure{}, cm.Failures...),
								Unmet:    append([]CoverageUnmet{}, cm.Unmet...),
								Met:      append([]CoverageMet{}, cm.Met...),
								Skipped:  append([]CoverageSkip{}, cm.Skipped...),
								Timings:  append(CoverageTimings{}, cm.Timings...),
							},
						}
					}
				}
			}
		}
	}
}

func (c *Coverage) SpecCoverage() (*SpecCoverage, error) {
	const root = "/"
	if c.OAS == nil {
		return nil, errors.New("spec not supplied")
	}
	result := newSpecCoverage()
	seenPaths := make(map[string]struct{})
	seenPath := func(tPaths map[string]struct{}) {
		for tPath := range tPaths {
			seenPaths[tPath] = struct{}{}
		}
	}
	if len(c.OAS.Methods) > 0 {
		rootCov := newSpecPathCoverage(root, nil)
		if tPaths, ok := c.normalizedPaths[root]; ok {
			seenPath(tPaths)
			result.CoveredPaths[root] = rootCov
			c.checkMethodCoverage(rootCov, c.OAS.Methods, tPaths)
		} else {
			seenPath(tPaths)
			result.NonCoveredPaths[root] = rootCov
		}
	}
	// covered paths...
	_ = c.OAS.WalkPaths(func(path string, pathDef *chioas.Path) (cont bool, err error) {
		if len(pathDef.Methods) > 0 {
			nPath := normalizePath(path)
			pathCov := newSpecPathCoverage(path, pathDef)
			if tPaths, ok := c.normalizedPaths[nPath]; ok {
				seenPath(tPaths)
				result.CoveredPaths[path] = pathCov
				c.checkMethodCoverage(pathCov, pathDef.Methods, tPaths)
			} else {
				seenPath(tPaths)
				result.NonCoveredPaths[path] = pathCov
			}
		}
		return true, nil
	})
	// unknown paths...
	for p, ce := range c.Endpoints {
		if _, ok := seenPaths[p]; !ok {
			pathCov := newSpecPathCoverage(p, nil)
			result.UnknownPaths[p] = pathCov
			for m, cm := range ce.Methods {
				pathCov.UnknownMethods[m] = &SpecMethodCoverage{
					Method: m,
					CoverageCommon: CoverageCommon{
						Failures: append([]CoverageFailure{}, cm.Failures...),
						Unmet:    append([]CoverageUnmet{}, cm.Unmet...),
						Met:      append([]CoverageMet{}, cm.Met...),
						Skipped:  append([]CoverageSkip{}, cm.Skipped...),
						Timings:  append(CoverageTimings{}, cm.Timings...),
					},
				}
			}
		}
	}
	return result, nil
}

type CoverageTiming struct {
	Endpoint Endpoint_
	Method   Method_
	Request  *http.Request
	Duration time.Duration
}

type CoverageTimings []CoverageTiming

type TimingStats struct {
	// Mean is the mean average response time. Gives a sense of typical latency under the current test conditions
	Mean time.Duration
	// StdDev is the standard deviation - how much individual response times deviate from the mean.
	//
	// A small StdDev means responses are consistent; a large one means erratic latency
	StdDev time.Duration // sqrt(Variance)
	// Variance is how much response times vary - in squared seconds.
	//
	// The lower the value, the better.  A high value indicates jitter and inconsistent response times
	Variance float64 // in seconds²
	// Minimum is the fastest observed response time. Indicates best-case performance
	Minimum time.Duration
	// Maximum is the slowest observed response time. Often reveals worst-case outliers (e.g., cold starts, timeouts)
	Maximum time.Duration
	// P50 is the median response time - half the runs were faster, half slower.
	//
	// Unlike Mean, it’s robust to outliers.  Indicates “Typical” latency.
	P50 time.Duration
	// P90 is the 90th percentile of response times - 9 out of 10 runs were this fast or faster.
	//
	// Useful for SLO/SLA checks.
	P90 time.Duration
	// P99 is the 99th percentile of response times - highlights rare but serious tail latencies.
	P99 time.Duration
	// Count is how many timings the stats are based on.
	//
	// Important for trustworthiness: low counts mean less reliable percentiles.
	Count int
	// Sample is whether the Variance / StdDev were computed using sample (n-1) or population (n) denominator — mostly for internal correctness.
	Sample bool
}

const sec = float64(time.Second)

func (ct CoverageTimings) Stats(sample bool) (TimingStats, bool) {
	if len(ct) == 0 {
		return TimingStats{}, false
	}
	n := 0.0
	meanSec := 0.0
	m2Sec2 := 0.0
	minD := ct[0].Duration
	maxD := ct[0].Duration
	durations := make([]time.Duration, len(ct))
	for i, d := range ct {
		durations[i] = d.Duration
	}
	for _, d := range durations {
		if d < minD {
			minD = d
		}
		if d > maxD {
			maxD = d
		}
		x := float64(d) / sec
		n++
		delta := x - meanSec
		meanSec += delta / n
		m2Sec2 += delta * (x - meanSec)
	}
	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})
	percentile := func(p float64) time.Duration {
		if p <= 0 {
			return durations[0]
		}
		pos := p * float64(len(durations)-1)
		i := int(math.Floor(pos))
		j := i + 1
		if j >= len(durations) {
			return durations[i]
		}
		f := pos - float64(i)
		// linear interpolation in nanoseconds (int64) with rounding
		a := float64(durations[i])
		b := float64(durations[j])
		return time.Duration(math.Round(a + f*(b-a)))
	}
	result := TimingStats{
		Sample:  sample,
		Mean:    time.Duration(math.Round(meanSec * sec)),
		Count:   len(ct),
		Maximum: maxD,
		Minimum: minD,
		P50:     percentile(0.5),
		P90:     percentile(0.9),
		P99:     percentile(0.99),
	}
	if sample {
		if n < 2 {
			return result, true
		}
		result.Variance = m2Sec2 / (n - 1.0)
	} else {
		result.Variance = m2Sec2 / n
	}
	result.StdDev = time.Duration(math.Round(math.Sqrt(result.Variance) * sec))
	return result, true
}

// Outliers returns the subset of timings whose Duration is at or above
// the given percentile threshold (upper-tail). For example, p=0.99 returns
// the slowest ~1% of requests. If timings is empty, returns nil.
// Percentile is clamped into [0,1]. p==0 returns all, p==1 returns only the max(es).
//
// Returned timings are sorted - slowest appearing last
func (ct CoverageTimings) Outliers(percentile float64) []CoverageTiming {
	if len(ct) == 0 {
		return nil
	}
	durs := make([]CoverageTiming, len(ct))
	copy(durs, ct)
	sort.Slice(durs, func(i, j int) bool {
		return durs[i].Duration < durs[j].Duration
	})
	// clamp percentile to max 1, or all for <= 0
	if math.IsNaN(percentile) || math.IsInf(percentile, 0) || percentile <= 0 {
		return durs
	} else if percentile > 1 {
		percentile = 1
	}
	pos := percentile * float64(len(durs)-1)
	i := int(math.Floor(pos))
	j := i + 1
	var start int
	var threshold time.Duration
	if j >= len(durs) {
		threshold = durs[i].Duration
		start = len(durs) - 1
	} else {
		f := pos - float64(i)
		a := float64(durs[i].Duration)
		b := float64(durs[j].Duration)
		threshold = time.Duration(math.Round(a + f*(b-a)))
		start = j
	}
	for bk := start - 1; bk >= 0 && durs[bk].Duration >= threshold; bk-- {
		start--
	}
	return durs[start:]
}

type nullCoverage struct{}

var _ CoverageCollector = (*nullCoverage)(nil)

func (n *nullCoverage) LoadSpec(r io.Reader) (err error) {
	// nullCoverage does nothing
	return nil
}

func (n *nullCoverage) ReportFailure(endpoint Endpoint_, method Method_, req *http.Request, err error) {
	// nullCoverage does nothing
}

func (n *nullCoverage) ReportUnmet(endpoint Endpoint_, method Method_, req *http.Request, exp Expectation, err error) {
	// nullCoverage does nothing
}

func (n *nullCoverage) ReportMet(endpoint Endpoint_, method Method_, req *http.Request, exp Expectation) {
	// nullCoverage does nothing
}

func (n *nullCoverage) ReportSkipped(endpoint Endpoint_, method Method_, req *http.Request, exp Expectation) {
	// nullCoverage does nothing
}

func (n *nullCoverage) ReportTiming(endpoint Endpoint_, method Method_, req *http.Request, dur time.Duration) {
	// nullCoverage does nothing
}
