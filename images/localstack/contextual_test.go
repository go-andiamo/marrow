package localstack

import (
	"bytes"
	"github.com/go-andiamo/marrow"
	"github.com/go-andiamo/marrow/common"
	"github.com/go-andiamo/marrow/coverage"
	"github.com/go-andiamo/marrow/with"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"testing"
)

func TestResolvablesAndBeforeAfters(t *testing.T) {
	do := &dummyDo{
		status: http.StatusOK,
		body:   []byte(`{"foo":"bar"}`),
	}
	options := Options{
		Services: Services{All},
		Dynamo:   testDynamoOptions,
	}
	endpoint := marrow.Endpoint("/api", "",
		marrow.Method("GET", "").AssertOK().
			SetVar(marrow.Before, "initial-count", DynamoItemsCount("TestTable")).
			Capture(DynamoPutItem(marrow.Before, "TestTable", marrow.JSON{"code": "foo", "value": "bar"})).
			SetVar(marrow.Before, "item", DynamoGetItem("TestTable", "code", "foo")).
			AssertEqual("bar", marrow.JsonPath(marrow.Var("item"), "value")).
			AssertEqual(marrow.Var("initial-count"), 0).
			AssertGreaterThan(DynamoItemsCount("TestTable"), marrow.Var("initial-count")).
			Capture(DynamoDeleteItem(marrow.After, "TestTable", "code", "foo")),
		marrow.Method("GET", "again").AssertOK().
			AssertEqual(0, DynamoItemsCount("TestTable")),
	)
	var cov *coverage.Coverage
	s := marrow.Suite(endpoint).Init(
		With(options),
		with.HttpDo(do),
		with.ReportCoverage(func(coverage *coverage.Coverage) {
			cov = coverage
		}),
	)
	err := s.Run()
	require.NoError(t, err)
	assert.Len(t, cov.Failures, 0)
	assert.Len(t, cov.Unmet, 0)
}

type dummyDo struct {
	status int
	body   []byte
	err    error
}

var _ common.HttpDo = (*dummyDo)(nil)

func (d *dummyDo) Do(req *http.Request) (*http.Response, error) {
	if d.err != nil {
		return nil, d.err
	}
	return &http.Response{
		StatusCode: d.status,
		Body:       io.NopCloser(bytes.NewReader(d.body)),
	}, nil
}
