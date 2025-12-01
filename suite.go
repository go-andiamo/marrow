package marrow

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/go-andiamo/marrow/common"
	"github.com/go-andiamo/marrow/coverage"
	"github.com/go-andiamo/marrow/mocks/service"
	htesting "github.com/go-andiamo/marrow/testing"
	"github.com/go-andiamo/marrow/with"
	"io"
	"maps"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// Suite_ is the interface for the main API tests runner
type Suite_ interface {
	// Init initializes the Suite using the provided withs
	//
	// a with can be from any provided by the [with] package -
	// with.ApiImage, with.Make, with.Database, with.HttpDo, with.ApiHost, with.Testing, with.Var, with.Cookie, with.ReportCoverage, with.CoverageCollector, with.OAS, with.Repeats, with.Logging
	//
	// a with can also be supporting image - see [images] package
	//
	// Note: Init takes a clone of the Suite, so that multiple runners with independent initialisation can be run from the same test endpoint definitions
	Init(withs ...with.With) Suite_
	// Run runs the test suite
	//
	// only critical errors are returned - unmet expectations are not (use coverage to inspect test failures and unmet expectations)
	Run() error
}

// Suite instantiates a new test suite for the provided Endpoint's
func Suite(endpoints ...Endpoint_) Suite_ {
	return &suite{
		endpoints:    endpoints,
		dbs:          namedDatabases{},
		vars:         make(map[Var]any),
		cookies:      make(map[string]*http.Cookie),
		mockServices: make(map[string]service.MockedService),
		images:       make(map[string]with.Image),
	}
}

type suite struct {
	endpoints     []Endpoint_
	withs         []with.With
	dbs           namedDatabases
	host          string
	port          int
	httpDo        common.HttpDo
	traceTimings  bool
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
	shutdowns     []func()
	mockServices  map[string]service.MockedService
	images        map[string]with.Image
	apiImage      with.ImageApi
	mutex         sync.RWMutex
}

var _ Suite_ = (*suite)(nil)
var _ with.SuiteInit = (*suite)(nil)

func (s *suite) AddDb(dbName string, db *sql.DB, dbArgs common.DatabaseArgs) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.dbs.register(dbName, db, dbArgs)
}

func (s *suite) SetHttpDo(do common.HttpDo) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.httpDo = do
}

func (s *suite) SetApiHost(host string, port int) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.host = host
	s.port = port
}

func (s *suite) SetTesting(t *testing.T) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.testing = t
}

func (s *suite) SetVar(name string, value any) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.vars[Var(name)] = value
}

func (s *suite) SetCookie(cookie *http.Cookie) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if cookie != nil {
		s.cookies[cookie.Name] = cookie
	}
}

func (s *suite) SetReportCoverage(fn func(coverage *coverage.Coverage)) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.reportCov = fn
}

func (s *suite) SetCoverageCollector(collector coverage.Collector) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if collector != nil {
		s.covCollector = collector
	}
}

func (s *suite) SetOAS(r io.Reader) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.oasReader = r
}

func (s *suite) SetRepeats(n int, stopOnFailure bool, resets ...func()) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.repeats = n
	s.repeatResets = resets
	s.stopOnFailure = stopOnFailure
}

func (s *suite) SetLogging(stdout io.Writer, stderr io.Writer) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.stdout = stdout
	s.stderr = stderr
}

func (s *suite) AddMockService(mock service.MockedService) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if mock != nil {
		s.mockServices[mock.Name()] = mock
	}
}

func (s *suite) AddSupportingImage(info with.Image) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if api, ok := info.(with.ImageApi); ok && api.IsApi() {
		s.apiImage = api
		// don't add api image to named images...
		return
	}
	name := info.Name()
	if _, ok := s.images[name]; ok {
		for idx := 2; ; idx++ {
			k := name + "-" + strconv.Itoa(idx)
			if _, exists := s.images[k]; !exists {
				name = k
				break
			}
		}
	}
	s.images[name] = info
}

func (s *suite) SetTraceTimings(collect bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.traceTimings = collect
}

func (s *suite) ResolveEnv(v any) (string, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	switch vt := v.(type) {
	case Var:
		if av, ok := s.vars[vt]; ok {
			return fmt.Sprintf("%v", av), nil
		} else {
			return "", fmt.Errorf("variable %q not found", string(vt))
		}
	case string:
		if av, err := s.resolveEnvString(vt); err == nil {
			return av, nil
		} else {
			return "", err
		}
	default:
		return fmt.Sprintf("%v", v), nil
	}
}

func (s *suite) resolveEnvString(str string) (string, error) {
	if !strings.Contains(str, "{$") {
		return str, nil
	}
	var b strings.Builder
	unresolved := false
	for i := 0; i < len(str); {
		j := strings.Index(str[i:], "{$")
		if j < 0 {
			b.WriteString(str[i:])
			break
		}
		j += i
		// count preceding backslashes...
		k := j - 1
		backslashes := 0
		for k >= i && str[k] == '\\' {
			backslashes++
			k--
		}
		// write the chunk before the backslashes...
		pre := j - backslashes
		b.WriteString(str[i:pre])
		if backslashes%2 == 1 {
			// escaped: keep one backslash as escape consumer, output "{$" literally
			// write the remaining backslashes (odd -> one fewer gets consumed)
			if backslashes > 1 {
				b.WriteString(str[pre : pre+backslashes-1])
			}
			b.WriteString("{$")
			i = j + 2
			continue
		}
		// unescaped placeholder: try to parse {$name}
		// scan name
		nameStart := j + 2
		nameEnd := nameStart
		for nameEnd < len(str) {
			c := str[nameEnd]
			if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' || c == ':' {
				nameEnd++
				continue
			}
			break
		}
		if nameEnd == nameStart || nameEnd >= len(str) || str[nameEnd] != '}' {
			// malformed token; treat literally
			b.WriteString("{$")
			i = nameStart
			continue
		}
		name := str[nameStart:nameEnd]
		parts := strings.Split(name, ":")
		found := false
		switch {
		case len(parts) == 1:
			if av, ok := s.vars[Var(name)]; ok {
				found = true
				b.WriteString(fmt.Sprintf("%v", av))
			}
		case len(parts) == 3 && parts[0] == "mock" && (parts[2] == "host" || parts[2] == "port"):
			// mock:name:port mock:name:host ... where name is the service name of a mock in s.mockServices
			if m, ok := s.mockServices[parts[1]]; ok {
				found = true
				if parts[2] == "host" {
					b.WriteString(m.ActualHost())
				} else {
					b.WriteString(fmt.Sprintf("%v", m.Port()))
				}
			}
		case len(parts) == 2 && parts[0] == "env":
			if s, ok := os.LookupEnv(parts[1]); ok {
				found = true
				b.WriteString(s)
			}
		case len(parts) > 1:
			// name:port name:host name:username name:password ... where name is the name of a supporting image in s.images
			if img, ok := s.images[parts[0]]; ok {
				switch parts[1] {
				case "host":
					found = true
					b.WriteString(img.Host())
				case "port":
					found = true
					b.WriteString(img.Port())
				case "mport":
					found = true
					b.WriteString(img.MappedPort())
				case "username":
					found = true
					b.WriteString(img.Username())
				case "password":
					found = true
					b.WriteString(img.Password())
				default:
					if ire, ok := img.(with.ImageResolveEnv); ok {
						tokens := parts[1:]
						if v, ok := ire.ResolveEnv(tokens...); ok {
							found = true
							b.WriteString(v)
						}
					}
				}
			}
		}
		if !found {
			// leave placeholder as-is and flag unresolved
			b.WriteString(str[j : nameEnd+1])
			unresolved = true
		}
		i = nameEnd + 1
	}
	if unresolved {
		return "", fmt.Errorf("unresolved markers/variables in string %q", b.String())
	}
	return b.String(), nil
}

func (s *suite) Init(withs ...with.With) Suite_ {
	return &suite{
		endpoints:    s.endpoints,
		dbs:          namedDatabases{},
		withs:        append(s.withs, withs...),
		vars:         make(map[Var]any),
		cookies:      make(map[string]*http.Cookie),
		mockServices: make(map[string]service.MockedService),
		images:       make(map[string]with.Image),
	}
}

func (s *suite) runInits() error {
	s.shutdowns = make([]func(), 0)
	s.stdout = os.Stdout
	s.stderr = os.Stderr
	supporting := make([]with.With, 0, len(s.withs))
	finals := make([]with.With, 0, len(s.withs))
	for _, w := range s.withs {
		if w != nil {
			switch w.Stage() {
			case with.Supporting:
				supporting = append(supporting, w)
			case with.Final:
				finals = append(finals, w)
			default:
				if err := w.Init(s); err != nil {
					return err
				} else if sdfn := w.Shutdown(); sdfn != nil {
					s.shutdowns = append(s.shutdowns, sdfn)
				}
			}
		}
	}
	track := &initTrack{}
	track.add(len(supporting))
	for _, w := range supporting {
		uw := w
		go func() {
			if !track.errored() {
				track.addShutdown(uw.Shutdown())
				if err := uw.Init(s); err != nil {
					track.error(err)
				}
			}
			track.done()
		}()
	}
	if err := track.wait(); err != nil {
		return err
	}
	track.add(len(finals))
	for _, w := range finals {
		uw := w
		go func() {
			if !track.errored() {
				track.addShutdown(uw.Shutdown())
				if err := uw.Init(s); err != nil {
					track.error(err)
				}
			}
			track.done()
		}()
	}
	if err := track.wait(); err != nil {
		return err
	}
	s.shutdowns = append(s.shutdowns, track.shutdowns...)
	return nil
}

func (s *suite) Run() error {
	if err := s.runInits(); err != nil {
		return err
	}
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
	ctx.stopListeners()
	for _, sdfn := range s.shutdowns {
		sdfn()
	}
	return nil
}

func (s *suite) initContext(cov coverage.Collector, t htesting.Helper) *context {
	result := newContext()
	result.coverage = cov
	result.traceTimings = s.traceTimings
	if s.httpDo != nil {
		result.httpDo = s.httpDo
	}
	host := s.host
	if host == "" {
		host = "localhost"
	}
	result.host = fmt.Sprintf("http://%s:%d", host, s.port)
	result.dbs = maps.Clone(s.dbs)
	result.images = maps.Clone(s.images)
	result.apiImage = s.apiImage
	result.testing = t
	result.mockServices = maps.Clone(s.mockServices)
	for k, v := range s.vars {
		result.vars[k] = v
	}
	for k, v := range s.cookies {
		result.cookieJar[k] = v
	}
	return result
}

type initTrack struct {
	shutdowns []func()
	err       error
	wg        sync.WaitGroup
	mu        sync.RWMutex
}

func (i *initTrack) addShutdown(sdfn func()) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if sdfn != nil {
		i.shutdowns = append(i.shutdowns, sdfn)
	}
}

func (i *initTrack) error(err error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if err != nil {
		i.err = err
	}
}

func (i *initTrack) errored() bool {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.err != nil
}

func (i *initTrack) add(delta int) {
	i.wg.Add(delta)
}

func (i *initTrack) done() {
	i.wg.Done()
}

func (i *initTrack) wait() error {
	i.wg.Wait()
	if i.err != nil {
		for _, sdfn := range i.shutdowns {
			sdfn()
		}
	}
	return i.err
}
