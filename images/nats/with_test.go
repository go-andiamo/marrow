package nats

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
	"sync"
	"testing"
)

func TestWithInit_Mocked(t *testing.T) {
	w := With(Options{})
	init := newMockInit()
	init.wg.Add(1)
	err := w.Init(init)
	require.NoError(t, err)
	init.wg.Wait()

	// check internals were set...
	assert.NotNil(t, w.Container())
	assert.NotEqual(t, "", w.MappedPort())
	assert.NotNil(t, w.Client())
	// check can't be started or init'd once already initialised...
	assert.Error(t, w.Start())
	assert.Error(t, w.Init(init))

	// check suit init was called
	assert.Len(t, init.called, 1)
	_, ok := init.called["AddSupportingImage:nats"]
	assert.True(t, ok)
	assert.Len(t, init.images, 1)
	img, ok := init.images[imageName]
	assert.True(t, ok)
	assert.Equal(t, imageName, img.Name())
	assert.True(t, img.IsDocker())
	assert.Equal(t, "localhost", img.Host())
	assert.Equal(t, defaultClientPort, img.Port())
	assert.NotEqual(t, defaultClientPort, img.MappedPort())
	assert.Equal(t, "nats", img.Username())
	assert.Equal(t, "nats", img.Password())

	w.Shutdown()()
}

func TestWithInit_Suite(t *testing.T) {
	w := With(Options{})
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
	mutex    sync.Mutex
	wg       sync.WaitGroup
	called   map[string]struct{}
	services map[string]service.MockedService
	images   map[string]with.Image
}

var _ with.SuiteInit = (*mockInit)(nil)

func (d *mockInit) AddDb(dnName string, db *sql.DB, dbArgs common.DatabaseArgs) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.called["AddDb"] = struct{}{}
}

func (d *mockInit) SetHttpDo(do common.HttpDo) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.called["SetHttpDo"] = struct{}{}
}

func (d *mockInit) SetApiHost(host string, port int) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.called["SetApiHost"] = struct{}{}
}

func (d *mockInit) SetTesting(t *testing.T) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.called["SetTesting"] = struct{}{}
}

func (d *mockInit) SetVar(name string, value any) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.called["SetVar"] = struct{}{}
}

func (d *mockInit) SetCookie(cookie *http.Cookie) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.called["SetCookie"] = struct{}{}
}

func (d *mockInit) SetReportCoverage(fn func(coverage *coverage.Coverage)) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.called["SetReportCoverage"] = struct{}{}
}

func (d *mockInit) SetCoverageCollector(collector coverage.Collector) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.called["SetCoverageCollector"] = struct{}{}
}

func (d *mockInit) SetOAS(r io.Reader) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.called["SetOAS"] = struct{}{}
}

func (d *mockInit) SetRepeats(n int, stopOnFailure bool, resets ...func()) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.called["SetRepeats"] = struct{}{}
}

func (d *mockInit) SetLogging(stdout io.Writer, stderr io.Writer) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.called["SetLogging"] = struct{}{}
}

func (d *mockInit) AddMockService(mock service.MockedService) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.called["AddMockService:"+mock.Name()] = struct{}{}
	d.services[mock.Name()] = mock
}

func (d *mockInit) AddSupportingImage(info with.Image) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.wg.Done()
	d.called["AddSupportingImage:"+info.Name()] = struct{}{}
	d.images[info.Name()] = info
}

func (d *mockInit) ResolveEnv(v any) (string, error) {
	return fmt.Sprintf("%v", v), nil
}

func (d *mockInit) SetTraceTimings(collect bool) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.called["SetTraceTimings"] = struct{}{}
}
