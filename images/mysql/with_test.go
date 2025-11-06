package mysql

import (
	"database/sql"
	"fmt"
	"github.com/go-andiamo/marrow"
	"github.com/go-andiamo/marrow/common"
	"github.com/go-andiamo/marrow/coverage"
	"github.com/go-andiamo/marrow/mocks/service"
	"github.com/go-andiamo/marrow/with"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"testing"
)

func TestWithDbImage(t *testing.T) {
	w := With("", Options{})
	assert.Equal(t, with.Supporting, w.Stage())
}

func TestWithInit_Mocked(t *testing.T) {
	w := With("", Options{})
	init := newMockInit()
	err := w.Init(init)
	require.NoError(t, err)

	// check internals were set...
	assert.NotNil(t, w.Database())
	assert.NotNil(t, w.Container())
	assert.NotEqual(t, "", w.MappedPort())
	// check can't be started or init'd once already initialised...
	assert.Error(t, w.Start())
	assert.Error(t, w.Init(init))

	// check suit init was called
	assert.Len(t, init.called, 2)
	_, ok := init.called["AddDb"]
	assert.True(t, ok)
	_, ok = init.called["AddSupportingImage:mysql"]
	assert.True(t, ok)
	assert.Len(t, init.images, 1)
	img, ok := init.images["mysql"]
	assert.True(t, ok)
	assert.Equal(t, "mysql", img.Name())
	assert.True(t, img.IsDocker())
	assert.Equal(t, "localhost", img.Host())
	assert.Equal(t, defaultPort, img.Port())
	assert.NotEqual(t, defaultPort, img.MappedPort())
	assert.Equal(t, defaultRootUsername, img.Username())
	assert.Equal(t, defaultRootPassword, img.Password())

	w.Shutdown()()
}

func TestWithInit_Suite(t *testing.T) {
	w := With("", Options{})
	s := marrow.Suite().Init(w)
	err := s.Run()
	require.NoError(t, err)
}

func newMockInit() *mockInit {
	return &mockInit{
		called:   make(map[string]struct{}),
		services: make(map[string]service.MockedService),
		images:   make(map[string]with.Image),
	}
}

type mockInit struct {
	called   map[string]struct{}
	services map[string]service.MockedService
	images   map[string]with.Image
}

var _ with.SuiteInit = (*mockInit)(nil)

func (d *mockInit) AddDb(dnName string, db *sql.DB, dbArgs common.DatabaseArgs) {
	d.called["AddDb"] = struct{}{}
}

func (d *mockInit) SetHttpDo(do common.HttpDo) {
	d.called["SetHttpDo"] = struct{}{}
}

func (d *mockInit) SetApiHost(host string, port int) {
	d.called["SetApiHost"] = struct{}{}
}

func (d *mockInit) SetTesting(t *testing.T) {
	d.called["SetTesting"] = struct{}{}
}

func (d *mockInit) SetVar(name string, value any) {
	d.called["SetVar"] = struct{}{}
}

func (d *mockInit) SetCookie(cookie *http.Cookie) {
	d.called["SetCookie"] = struct{}{}
}

func (d *mockInit) SetReportCoverage(fn func(coverage *coverage.Coverage)) {
	d.called["SetReportCoverage"] = struct{}{}
}

func (d *mockInit) SetCoverageCollector(collector coverage.Collector) {
	d.called["SetCoverageCollector"] = struct{}{}
}

func (d *mockInit) SetOAS(r io.Reader) {
	d.called["SetOAS"] = struct{}{}
}

func (d *mockInit) SetRepeats(n int, stopOnFailure bool, resets ...func()) {
	d.called["SetRepeats"] = struct{}{}
}

func (d *mockInit) SetLogging(stdout io.Writer, stderr io.Writer) {
	d.called["SetLogging"] = struct{}{}
}

func (d *mockInit) AddMockService(mock service.MockedService) {
	d.called["AddMockService:"+mock.Name()] = struct{}{}
	d.services[mock.Name()] = mock
}

func (d *mockInit) AddSupportingImage(info with.Image) {
	d.called["AddSupportingImage:"+info.Name()] = struct{}{}
	d.images[info.Name()] = info
}

func (d *mockInit) ResolveEnv(v any) (string, error) {
	return fmt.Sprintf("%v", v), nil
}

func (d *mockInit) SetTraceTimings(collect bool) {
	d.called["SetTraceTimings"] = struct{}{}
}
