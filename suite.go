package marrow

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/go-andiamo/marrow/coverage"
	"io"
	"maps"
	"net/http"
	"testing"
)

type Suite_ interface {
	Init(withs ...With) Suite_
	InitFunc(func(init SuiteInit)) Suite_
	Run() error
}

func Suite(endpoints ...Endpoint_) Suite_ {
	return &suite{
		endpoints: endpoints,
		vars:      make(map[Var]any),
		cookies:   make(map[string]*http.Cookie),
	}
}

type suite struct {
	endpoints     []Endpoint_
	withs         []With
	db            *sql.DB
	dbArgMarkers  DatabaseArgMarkers
	host          string
	port          int
	httpDo        HttpDo
	testing       *testing.T
	vars          map[Var]any
	cookies       map[string]*http.Cookie
	reportCov     func(*coverage.Coverage)
	covCollector  coverage.Collector
	oasReader     io.Reader
	repeats       int
	repeatResets  []func(si SuiteInit)
	stopOnFailure bool
}

func (s *suite) SetDb(db *sql.DB) {
	s.db = db
}

func (s *suite) SetDbArgMarkers(dbArgMarkers DatabaseArgMarkers) {
	s.dbArgMarkers = dbArgMarkers
}

func (s *suite) SetHttpDo(do HttpDo) {
	s.httpDo = do
}

func (s *suite) SetApiHost(host string, port int) {
	s.host = host
	s.port = port
}

func (s *suite) SetApiImage(image string, more ...any) {
	//TODO implement me
	panic("implement me")
}

func (s *suite) SetTesting(t *testing.T) {
	s.testing = t
}

func (s *suite) SetVar(name string, value any) {
	s.vars[Var(name)] = value
}

func (s *suite) SetCookie(cookie *http.Cookie) {
	if cookie != nil {
		s.cookies[cookie.Name] = cookie
	}
}

func (s *suite) SetReportCoverage(fn func(coverage *coverage.Coverage)) {
	s.reportCov = fn
}

func (s *suite) SetCoverageCollector(collector coverage.Collector) {
	if collector != nil {
		s.covCollector = collector
	}
}

func (s *suite) SetOAS(r io.Reader) {
	s.oasReader = r
}

func (s *suite) SetRepeats(n int, stopOnFailure bool, resets ...func(si SuiteInit)) {
	s.repeats = n
	s.repeatResets = resets
	s.stopOnFailure = stopOnFailure
}

func (s *suite) Init(withs ...With) Suite_ {
	s.withs = append(s.withs, withs...)
	return s
}

func (s *suite) InitFunc(fn func(init SuiteInit)) Suite_ {
	if fn != nil {
		s.withs = append(s.withs, &with{fn: fn})
	}
	return s
}

func (s *suite) Run() error {
	for _, w := range s.withs {
		if w != nil {
			w.Init(s)
		}
	}
	do := s.httpDo
	if do == nil {
		do = http.DefaultClient
	}
	host := s.host
	if host == "" {
		host = "localhost"
	}
	var cov coverage.Collector = coverage.NewNullCoverage()
	if s.covCollector != nil {
		cov = s.covCollector
	}
	var actualCov *coverage.Coverage
	if s.reportCov != nil {
		if s.covCollector != nil {
			return errors.New("cannot report coverage with custom coverage collector")
		}
		actualCov = coverage.NewCoverage()
		cov = actualCov
	}
	if s.oasReader != nil {
		if err := cov.LoadSpec(s.oasReader); err != nil {
			return err
		}
	}
	ctx := &context{
		coverage:     cov,
		httpDo:       do,
		host:         fmt.Sprintf("http://%s:%d", host, s.port),
		db:           s.db,
		dbArgMarkers: s.dbArgMarkers,
		testing:      s.testing,
		vars:         maps.Clone(s.vars),
		cookieJar:    maps.Clone(s.cookies),
	}
	for _, e := range s.endpoints {
		if !ctx.run(e.Url(), e) {
			break
		}
	}
	if s.repeats > 0 && (!s.stopOnFailure || !cov.HasFailures()) {
		ctx.vars = maps.Clone(s.vars)
		ctx.cookieJar = maps.Clone(s.cookies)
		ctx.testing = nil
		ctx.currTesting = nil
		for r := 0; r < s.repeats; r++ {
			for _, reset := range s.repeatResets {
				reset(s)
			}
			for _, e := range s.endpoints {
				if !ctx.run(e.Url(), e) {
					break
				}
			}
			if s.stopOnFailure && cov.HasFailures() {
				break
			}
		}
	}
	if s.reportCov != nil {
		s.reportCov(actualCov)
	}
	return nil
}
