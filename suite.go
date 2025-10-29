package marrow

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/go-andiamo/marrow/common"
	"github.com/go-andiamo/marrow/coverage"
	htesting "github.com/go-andiamo/marrow/testing"
	"github.com/go-andiamo/marrow/with"
	"io"
	"maps"
	"net/http"
	"os"
	"testing"
	"time"
)

type Suite_ interface {
	Init(withs ...with.With) Suite_
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
	withs         []with.With
	db            *sql.DB
	dbArgMarkers  common.DatabaseArgMarkers
	host          string
	port          int
	httpDo        common.HttpDo
	testing       *testing.T
	vars          map[Var]any
	cookies       map[string]*http.Cookie
	reportCov     func(*coverage.Coverage)
	covCollector  coverage.Collector
	oasReader     io.Reader
	repeats       int
	repeatResets  []func()
	stopOnFailure bool
	stdout        io.Writer
	stderr        io.Writer
}

func (s *suite) SetDb(db *sql.DB) {
	s.db = db
}

func (s *suite) SetDbArgMarkers(dbArgMarkers common.DatabaseArgMarkers) {
	s.dbArgMarkers = dbArgMarkers
}

func (s *suite) SetHttpDo(do common.HttpDo) {
	s.httpDo = do
}

func (s *suite) SetApiHost(host string, port int) {
	s.host = host
	s.port = port
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

func (s *suite) SetRepeats(n int, stopOnFailure bool, resets ...func()) {
	s.repeats = n
	s.repeatResets = resets
	s.stopOnFailure = stopOnFailure
}

func (s *suite) SetLogging(stdout io.Writer, stderr io.Writer) {
	s.stdout = stdout
	s.stderr = stderr
}

func (s *suite) Init(withs ...with.With) Suite_ {
	return &suite{
		endpoints: s.endpoints,
		withs:     append(s.withs, withs...),
		vars:      make(map[Var]any),
		cookies:   make(map[string]*http.Cookie),
	}
}

func (s *suite) runInits() {
	for _, w := range s.withs {
		if w != nil {
			w.Init(s)
		}
	}
}

func (s *suite) Run() error {
	s.runInits()
	cov := coverage.NewNullCoverage()
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
	if s.stdout == nil {
		s.stdout = os.Stdout
	}
	if s.stderr == nil {
		s.stderr = os.Stderr
	}
	t := htesting.NewHelper(s.testing, s.stdout, s.stderr)
	ctx := s.initContext(cov, t)
	for _, e := range s.endpoints {
		if !ctx.run(e.Url(), e) {
			break
		}
	}
	t.End()
	if s.repeats > 0 && (!s.stopOnFailure || !cov.HasFailures()) {
		_, _ = fmt.Fprintln(s.stdout, "")
		ctx.vars = maps.Clone(s.vars)
		ctx.cookieJar = maps.Clone(s.cookies)
		ctx.testing = nil
		ctx.currTesting = nil
		ctx.failed = false
		for r := 0; r < s.repeats; r++ {
			_, _ = fmt.Fprintf(s.stdout, ">>> REPEAT %d/%d\n", r+1, s.repeats)
			for _, reset := range s.repeatResets {
				reset()
			}
			start := time.Now()
			for _, e := range s.endpoints {
				if !ctx.run(e.Url(), e) {
					break
				}
			}
			if s.stopOnFailure && cov.HasFailures() {
				_, _ = fmt.Fprintf(s.stderr, "    FAILED (%s)\n", time.Since(start))
				break
			}
			_, _ = fmt.Fprintf(s.stderr, "    FINISHED (%s)\n", time.Since(start))
		}
		_, _ = fmt.Fprintln(s.stdout, "")
	}
	if s.reportCov != nil {
		s.reportCov(actualCov)
	}
	return nil
}

func (s *suite) initContext(cov coverage.Collector, t htesting.Helper) *context {
	result := newContext()
	result.coverage = cov
	if s.httpDo != nil {
		result.httpDo = s.httpDo
	}
	host := s.host
	if host == "" {
		host = "localhost"
	}
	result.host = fmt.Sprintf("http://%s:%d", host, s.port)
	result.db = s.db
	result.dbArgMarkers = s.dbArgMarkers
	result.testing = t
	for k, v := range s.vars {
		result.vars[k] = v
	}
	for k, v := range s.cookies {
		result.cookieJar[k] = v
	}
	return result
}
