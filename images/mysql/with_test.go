package mysql

import (
	"database/sql"
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
	w := WithDbImage(Options{})
	assert.Equal(t, with.Supporting, w.Stage())
}

func TestWithInit_Mocked(t *testing.T) {
	w := WithDbImage(Options{})
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
	assert.Len(t, init.called, 3)
	_, ok := init.called["SetDb"]
	assert.True(t, ok)
	_, ok = init.called["SetDbArgMarkers"]
	assert.True(t, ok)
	_, ok = init.called["AddSupportingImage:mysql"]
	assert.True(t, ok)
	assert.Len(t, init.images, 1)
	_, ok = init.images["mysql"]
	assert.True(t, ok)

	w.Shutdown()()
}

func TestWithInit_Suite(t *testing.T) {
	w := WithDbImage(Options{})
	s := marrow.Suite().Init(w)
	err := s.Run()
	require.NoError(t, err)
}

func newMockInit() *mockInit {
	return &mockInit{
		called:   make(map[string]struct{}),
		services: make(map[string]service.MockedService),
		images:   make(map[string]with.ImageInfo),
	}
}

type mockInit struct {
	called   map[string]struct{}
	services map[string]service.MockedService
	images   map[string]with.ImageInfo
}

var _ with.SuiteInit = (*mockInit)(nil)

func (d *mockInit) SetDb(db *sql.DB) {
	d.called["SetDb"] = struct{}{}
}

func (d *mockInit) SetDbArgMarkers(dbArgMarkers common.DatabaseArgMarkers) {
	d.called["SetDbArgMarkers"] = struct{}{}
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

func (d *mockInit) AddSupportingImage(info with.ImageInfo) {
	d.called["AddSupportingImage:"+info.Name] = struct{}{}
	d.images[info.Name] = info
}
