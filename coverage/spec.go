package coverage

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-andiamo/chioas"
	"gopkg.in/yaml.v3"
	"io"
)

type Spec struct {
	CoveredPaths    map[string]*SpecPath
	NonCoveredPaths map[string]*SpecPath
	UnknownPaths    map[string]*SpecPath
}

type SpecPath struct {
	Path              string
	PathDef           *chioas.Path
	CoveredMethods    map[string]*SpecMethod
	NonCoveredMethods map[string]*SpecMethod
	UnknownMethods    map[string]*SpecMethod
}

type SpecMethod struct {
	Method    string
	MethodDef *chioas.Method
	Common
}

func (s *Spec) PathsCovered() (total int, covered int, perc float64) {
	covered = len(s.CoveredPaths)
	total = covered + len(s.NonCoveredPaths)
	perc = float64(covered) / float64(total)
	return
}

func (s *Spec) MethodsCovered() (total int, covered int, perc float64) {
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

func newSpecCoverage() *Spec {
	return &Spec{
		CoveredPaths:    make(map[string]*SpecPath),
		NonCoveredPaths: make(map[string]*SpecPath),
		UnknownPaths:    make(map[string]*SpecPath),
	}
}

func newSpecPathCoverage(path string, def *chioas.Path) *SpecPath {
	return &SpecPath{
		Path:              path,
		PathDef:           def,
		CoveredMethods:    make(map[string]*SpecMethod),
		NonCoveredMethods: make(map[string]*SpecMethod),
		UnknownMethods:    make(map[string]*SpecMethod),
	}
}

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

func (c *Coverage) SpecCoverage() (*Spec, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
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
				pathCov.UnknownMethods[m] = &SpecMethod{
					Method: m,
					Common: Common{
						Failures: append([]Failure{}, cm.Failures...),
						Unmet:    append([]Unmet{}, cm.Unmet...),
						Met:      append([]Met{}, cm.Met...),
						Skipped:  append([]Skip{}, cm.Skipped...),
						Timings:  append(Timings{}, cm.Timings...),
					},
				}
			}
		}
	}
	return result, nil
}

func (c *Coverage) checkMethodCoverage(pathCov *SpecPath, methods chioas.Methods, tPaths map[string]struct{}) {
	for m, mDef := range methods {
		mCov := &SpecMethod{
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
						pathCov.UnknownMethods[m] = &SpecMethod{
							Method: m,
							Common: Common{
								Failures: append([]Failure{}, cm.Failures...),
								Unmet:    append([]Unmet{}, cm.Unmet...),
								Met:      append([]Met{}, cm.Met...),
								Skipped:  append([]Skip{}, cm.Skipped...),
								Timings:  append(Timings{}, cm.Timings...),
							},
						}
					}
				}
			}
		}
	}
}
