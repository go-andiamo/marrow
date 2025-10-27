package marrow

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/go-andiamo/marrow/coverage"
	htesting "github.com/go-andiamo/marrow/testing"
	"io"
	"maps"
	"net/http"
	"os"
	"testing"
	"time"
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
	stdout        io.Writer
	stderr        io.Writer
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

func (s *suite) SetLogging(stdout io.Writer, stderr io.Writer) {
	s.stdout = stdout
	s.stderr = stderr
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

func (s *suite) runInits() {
	for _, w := range s.withs {
		if w != nil {
			w.Init(s)
		}
	}
}

func (s *suite) Run() error {
	s.runInits()
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
	if s.stdout == nil {
		s.stdout = os.Stdout
	}
	if s.stderr == nil {
		s.stderr = os.Stderr
	}
	t := htesting.NewHelper(s.testing, s.stdout, s.stderr)
	ctx := &context{
		coverage:     cov,
		httpDo:       do,
		host:         fmt.Sprintf("http://%s:%d", host, s.port),
		db:           s.db,
		dbArgMarkers: s.dbArgMarkers,
		testing:      t,
		vars:         maps.Clone(s.vars),
		cookieJar:    maps.Clone(s.cookies),
	}
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
		for r := 0; r < s.repeats; r++ {
			_, _ = fmt.Fprintf(s.stdout, ">>> REPEAT %d/%d\n", r+1, s.repeats)
			for _, reset := range s.repeatResets {
				reset(s)
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
