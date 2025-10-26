package coverage

import (
	"bytes"
	"github.com/go-andiamo/chioas"
	"github.com/go-andiamo/marrow/common"
	"github.com/go-andiamo/marrow/framing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"os"
	"testing"
)

func TestCoverage_LoadSpec_Json(t *testing.T) {
	f, err := os.Open("../_examples/petstore.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	cov := NewCoverage()
	err = cov.LoadSpec(f)
	require.NoError(t, err)
	assert.NotNil(t, cov.OAS)
}

func TestCoverage_LoadSpec_Yaml(t *testing.T) {
	f, err := os.Open("../_examples/petstore.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	cov := NewCoverage()
	err = cov.LoadSpec(f)
	require.NoError(t, err)
	assert.NotNil(t, cov.OAS)
}

func TestCoverage_LoadSpec_Errors(t *testing.T) {
	r := bytes.NewReader([]byte(`{invalid json}`))

	cov := NewCoverage()
	err := cov.LoadSpec(r)
	require.Error(t, err)
}

func TestCoverage_SpecCoverage(t *testing.T) {
	f, err := os.Open("../_examples/petstore.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	cov := NewCoverage()
	err = cov.LoadSpec(f)
	require.NoError(t, err)

	// add some met endpoints/methods...
	cov.ReportMet(&testEndpoint{"/foos"}, &testMethod{"GET"}, nil, &testExpectation{"expected something"})
	cov.ReportMet(&testEndpoint{"/api/pets"}, &testMethod{"GET"}, nil, &testExpectation{"expected something"})
	cov.ReportMet(&testEndpoint{"/api/pets"}, &testMethod{"PUT"}, nil, &testExpectation{"expected something"})
	cov.ReportMet(&testEndpoint{"/api/pets/{pet_id}"}, &testMethod{"GET"}, nil, &testExpectation{"expected something"})
	cov.ReportMet(&testEndpoint{"/api/pets/{pet_id}"}, &testMethod{"PATCH"}, nil, &testExpectation{"expected something"})
	cov.ReportMet(&testEndpoint{"/api/pets/{id}"}, &testMethod{"PATCH"}, nil, &testExpectation{"expected something"})

	// get spec coverage...
	specCov, err := cov.SpecCoverage()
	require.NoError(t, err)
	require.NotNil(t, specCov)

	assert.Len(t, specCov.CoveredPaths, 2)
	assert.Len(t, specCov.NonCoveredPaths, 3)
	assert.Len(t, specCov.UnknownPaths, 1)

	total, covered, perc := specCov.PathsCovered()
	assert.Equal(t, 5, total)
	assert.Equal(t, 2, covered)
	assert.Equal(t, 0.4, perc)
	total, covered, perc = specCov.MethodsCovered()
	assert.Equal(t, 8, total)
	assert.Equal(t, 2, covered)
	assert.Equal(t, 0.25, perc)

	t.Run("with root method (not covered)", func(t *testing.T) {
		// forge oas root method...
		cov.OAS.Methods = chioas.Methods{
			http.MethodGet: {},
		}
		specCov, err = cov.SpecCoverage()
		require.NoError(t, err)
		require.NotNil(t, specCov)

		assert.Len(t, specCov.CoveredPaths, 2)
		assert.Len(t, specCov.NonCoveredPaths, 4)
		assert.Len(t, specCov.UnknownPaths, 1)
	})
	t.Run("with root method (covered)", func(t *testing.T) {
		// add root met...
		cov.ReportMet(&testEndpoint{"/"}, &testMethod{"GET"}, nil, &testExpectation{"expected something"})

		specCov, err = cov.SpecCoverage()
		require.NoError(t, err)
		require.NotNil(t, specCov)

		assert.Len(t, specCov.CoveredPaths, 3)
		assert.Len(t, specCov.NonCoveredPaths, 3)
		assert.Len(t, specCov.UnknownPaths, 1)
	})
}

func TestCoverage_SpecCoverage_ErrorsWithNoSpec(t *testing.T) {
	cov := NewCoverage()

	_, err := cov.SpecCoverage()
	require.Error(t, err)
}

type testEndpoint struct {
	url string
}

var _ common.Endpoint = (*testEndpoint)(nil)

func (t *testEndpoint) Url() string {
	return t.url
}

func (t *testEndpoint) Description() string {
	return ""
}

func (t *testEndpoint) Frame() *framing.Frame {
	return framing.NewFrame(0)
}

type testMethod struct {
	method string
}

var _ common.Method = (*testMethod)(nil)

func (t *testMethod) MethodName() string {
	return t.method
}

func (t *testMethod) Description() string {
	return ""
}

func (t *testMethod) Frame() *framing.Frame {
	return framing.NewFrame(0)
}

type testExpectation struct {
	name string
}

var _ common.Expectation = (*testExpectation)(nil)

func (t *testExpectation) Name() string {
	return t.name
}

func (t *testExpectation) Frame() *framing.Frame {
	return framing.NewFrame(0)
}
