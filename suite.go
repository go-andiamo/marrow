package marrow

import (
	"database/sql"
	"fmt"
	"net/http"
	"testing"
)

type Suite_ interface {
	Init(withs ...With) Suite_
	InitFunc(func(init SuiteInit)) Suite_
	Run()
}

func Suite(endpoints ...Endpoint_) Suite_ {
	return &suite{
		endpoints: endpoints,
		vars:      make(map[Var]any),
		cookies:   make(map[string]*http.Cookie),
	}
}

type suite struct {
	endpoints    []Endpoint_
	withs        []With
	db           *sql.DB
	dbArgMarkers DatabaseArgMarkers
	host         string
	port         int
	httpDo       HttpDo
	testing      *testing.T
	vars         map[Var]any
	cookies      map[string]*http.Cookie
	covCollect   func(*Coverage)
	//todo other fields... to match everything that can be set
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

func (s *suite) SetCoverageCollect(fn func(coverage *Coverage)) {
	s.covCollect = fn
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

func (s *suite) Run() {
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
	cov := newCoverage()
	ctx := &context{
		suite:        s,
		coverage:     cov,
		httpDo:       do,
		host:         fmt.Sprintf("http://%s:%d", host, s.port),
		db:           s.db,
		dbArgMarkers: s.dbArgMarkers,
		testing:      s.testing,
		vars:         s.vars,
		cookieJar:    s.cookies,
	}
	for _, e := range s.endpoints {
		if !ctx.run("Endpoint: "+e.Url()+" "+e.Description(), e) {
			break
		}
	}
	if s.covCollect != nil {
		s.covCollect(cov)
	}
}
