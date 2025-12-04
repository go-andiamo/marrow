package localstack

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
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
	w := With(Options{
		Services: Services{All},
		CustomServices: CustomServiceBuilders{
			customServiceBuild,
		},
	})
	init := newMockInit()
	init.wg.Add(7)

	err := w.Init(init)
	require.NoError(t, err)

	// check internals were set...
	assert.NotNil(t, w.Container())
	// check can't be started or init'd once already initialised...
	assert.Error(t, w.Start())
	assert.Error(t, w.Init(init))

	init.wg.Wait()

	// check suit init was called
	assert.Len(t, init.called, 7)
	assert.Len(t, init.images, 7)
	_, ok := init.called["AddSupportingImage:dynamo"]
	assert.True(t, ok)
	_, ok = init.called["AddSupportingImage:s3"]
	assert.True(t, ok)
	_, ok = init.called["AddSupportingImage:sns"]
	assert.True(t, ok)
	_, ok = init.called["AddSupportingImage:sqs"]
	assert.True(t, ok)
	_, ok = init.called["AddSupportingImage:secrets-service"]
	assert.True(t, ok)
	_, ok = init.called["AddSupportingImage:lambda"]
	assert.True(t, ok)
	_, ok = init.called["AddSupportingImage:custom"]
	assert.True(t, ok)

	w.Shutdown()()
}

func TestWithInit_Suite(t *testing.T) {
	w := With(Options{
		Services: Services{S3},
	})
	s := marrow.Suite().Init(w)
	err := s.Run()
	require.NoError(t, err)
}

func customServiceBuild(ctx context.Context, awsCfg aws.Config, host string, mappedPort string) (image with.Image, err error) {
	return &customService{
		host:       host,
		mappedPort: mappedPort,
	}, nil
}

type customService struct {
	host       string
	mappedPort string
}

var _ with.Image = (*customService)(nil)

func (c *customService) Name() string {
	return "custom"
}

func (c *customService) Host() string {
	return c.host
}

func (c *customService) Port() string {
	return defaultPort
}

func (c *customService) MappedPort() string {
	return c.mappedPort
}

func (c *customService) IsDocker() bool {
	return true
}

func (c *customService) Username() string {
	return ""
}

func (c *customService) Password() string {
	return ""
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
