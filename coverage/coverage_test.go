package coverage

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNullCoverage_ReportFailure(t *testing.T) {
	t.Run("top level", func(t *testing.T) {
		cov := NewCoverage()
		cov.ReportFailure(nil, nil, nil, nil)
		assert.True(t, cov.HasFailures())
		assert.Len(t, cov.Endpoints, 0)
		assert.Len(t, cov.Failures, 1)
	})
	t.Run("endpoint only", func(t *testing.T) {
		cov := NewCoverage()
		cov.ReportFailure(&testEndpoint{"/"}, nil, nil, nil)
		assert.True(t, cov.HasFailures())
		assert.Len(t, cov.Failures, 1)
		assert.Len(t, cov.Endpoints, 1)
		assert.Len(t, cov.Endpoints["/"].Failures, 1)
		assert.Len(t, cov.Endpoints["/"].Methods, 0)
	})
	t.Run("endpoint+method", func(t *testing.T) {
		cov := NewCoverage()
		cov.ReportFailure(&testEndpoint{"/"}, &testMethod{"GET"}, nil, nil)
		assert.True(t, cov.HasFailures())
		assert.Len(t, cov.Failures, 1)
		assert.Len(t, cov.Endpoints, 1)
		assert.Len(t, cov.Endpoints["/"].Failures, 1)
		assert.Len(t, cov.Endpoints["/"].Methods, 1)
		assert.Len(t, cov.Endpoints["/"].Methods["GET"].Failures, 1)
	})
}

func TestNullCoverage_ReportUnmet(t *testing.T) {
	t.Run("top level", func(t *testing.T) {
		cov := NewCoverage()
		cov.ReportUnmet(nil, nil, nil, nil, nil)
		assert.True(t, cov.HasFailures())
		assert.Len(t, cov.Endpoints, 0)
		assert.Len(t, cov.Unmet, 1)
	})
	t.Run("endpoint only", func(t *testing.T) {
		cov := NewCoverage()
		cov.ReportUnmet(&testEndpoint{"/"}, nil, nil, nil, nil)
		assert.True(t, cov.HasFailures())
		assert.Len(t, cov.Unmet, 1)
		assert.Len(t, cov.Endpoints, 1)
		assert.Len(t, cov.Endpoints["/"].Unmet, 1)
		assert.Len(t, cov.Endpoints["/"].Methods, 0)
	})
	t.Run("endpoint+method", func(t *testing.T) {
		cov := NewCoverage()
		cov.ReportUnmet(&testEndpoint{"/"}, &testMethod{"GET"}, nil, nil, nil)
		assert.True(t, cov.HasFailures())
		assert.Len(t, cov.Unmet, 1)
		assert.Len(t, cov.Endpoints, 1)
		assert.Len(t, cov.Endpoints["/"].Unmet, 1)
		assert.Len(t, cov.Endpoints["/"].Methods, 1)
		assert.Len(t, cov.Endpoints["/"].Methods["GET"].Unmet, 1)
	})
}

func TestNullCoverage_ReportMet(t *testing.T) {
	t.Run("top level", func(t *testing.T) {
		cov := NewCoverage()
		cov.ReportMet(nil, nil, nil, nil)
		assert.False(t, cov.HasFailures())
		assert.Len(t, cov.Endpoints, 0)
		assert.Len(t, cov.Met, 1)
	})
	t.Run("endpoint only", func(t *testing.T) {
		cov := NewCoverage()
		cov.ReportMet(&testEndpoint{"/"}, nil, nil, nil)
		assert.False(t, cov.HasFailures())
		assert.Len(t, cov.Met, 1)
		assert.Len(t, cov.Endpoints, 1)
		assert.Len(t, cov.Endpoints["/"].Met, 1)
		assert.Len(t, cov.Endpoints["/"].Methods, 0)
	})
	t.Run("endpoint+method", func(t *testing.T) {
		cov := NewCoverage()
		cov.ReportMet(&testEndpoint{"/"}, &testMethod{"GET"}, nil, nil)
		assert.False(t, cov.HasFailures())
		assert.Len(t, cov.Met, 1)
		assert.Len(t, cov.Endpoints, 1)
		assert.Len(t, cov.Endpoints["/"].Met, 1)
		assert.Len(t, cov.Endpoints["/"].Methods, 1)
		assert.Len(t, cov.Endpoints["/"].Methods["GET"].Met, 1)
	})
}

func TestNullCoverage_ReportSkipped(t *testing.T) {
	t.Run("top level", func(t *testing.T) {
		cov := NewCoverage()
		cov.ReportSkipped(nil, nil, nil, nil)
		assert.False(t, cov.HasFailures())
		assert.Len(t, cov.Endpoints, 0)
		assert.Len(t, cov.Skipped, 1)
	})
	t.Run("endpoint only", func(t *testing.T) {
		cov := NewCoverage()
		cov.ReportSkipped(&testEndpoint{"/"}, nil, nil, nil)
		assert.False(t, cov.HasFailures())
		assert.Len(t, cov.Skipped, 1)
		assert.Len(t, cov.Endpoints, 1)
		assert.Len(t, cov.Endpoints["/"].Skipped, 1)
		assert.Len(t, cov.Endpoints["/"].Methods, 0)
	})
	t.Run("endpoint+method", func(t *testing.T) {
		cov := NewCoverage()
		cov.ReportSkipped(&testEndpoint{"/"}, &testMethod{"GET"}, nil, nil)
		assert.False(t, cov.HasFailures())
		assert.Len(t, cov.Skipped, 1)
		assert.Len(t, cov.Endpoints, 1)
		assert.Len(t, cov.Endpoints["/"].Skipped, 1)
		assert.Len(t, cov.Endpoints["/"].Methods, 1)
		assert.Len(t, cov.Endpoints["/"].Methods["GET"].Skipped, 1)
	})
}

func TestNullCoverage_ReportTiming(t *testing.T) {
	t.Run("top level", func(t *testing.T) {
		cov := NewCoverage()
		cov.ReportTiming(nil, nil, nil, time.Second)
		assert.False(t, cov.HasFailures())
		assert.Len(t, cov.Endpoints, 0)
		assert.Len(t, cov.Timings, 1)
	})
	t.Run("endpoint only", func(t *testing.T) {
		cov := NewCoverage()
		cov.ReportTiming(&testEndpoint{"/"}, nil, nil, time.Second)
		assert.False(t, cov.HasFailures())
		assert.Len(t, cov.Timings, 1)
		assert.Len(t, cov.Endpoints, 1)
		assert.Len(t, cov.Endpoints["/"].Timings, 1)
		assert.Len(t, cov.Endpoints["/"].Methods, 0)
	})
	t.Run("endpoint+method", func(t *testing.T) {
		cov := NewCoverage()
		cov.ReportTiming(&testEndpoint{"/"}, &testMethod{"GET"}, nil, time.Second)
		assert.False(t, cov.HasFailures())
		assert.Len(t, cov.Timings, 1)
		assert.Len(t, cov.Endpoints, 1)
		assert.Len(t, cov.Endpoints["/"].Timings, 1)
		assert.Len(t, cov.Endpoints["/"].Methods, 1)
		assert.Len(t, cov.Endpoints["/"].Methods["GET"].Timings, 1)
	})
}

func Test_requestShallowClone(t *testing.T) {
	require.Nil(t, requestShallowClone(nil))
	req := httptest.NewRequest("GET", "/", strings.NewReader("{}"))
	req2 := requestShallowClone(req)
	assert.Equal(t, http.NoBody, req2.Body)
	br, err := req2.GetBody()
	assert.NoError(t, err)
	assert.Equal(t, http.NoBody, br)
}
