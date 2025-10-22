package marrow

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"testing"
)

func TestSuite(t *testing.T) {
	t.Skip()
	s := Suite(
		Endpoint("/foos", "Foos endpoint",
			//SetVar(Before, "db", Query("SELECT 1")),
			Method(GET, "Get foos").
				ExpectStatus(Var("OK")).
				ExpectOK().
				//SetVar(Before, "z", Query("xxx", Var("yyy"), Query("zzz"))).
				SetVar(After, "body", BodyPath(".")).
				SetVar(After, "foo", JsonPath(Var("body"), "foo")).
				ExpectEqual(Var("foo"), "xxx").
				ExpectEqual(JsonPath(Var("body"), "foo"), "xxx").
				ExpectEqual(JsonPath(Var("body"), "foo"), 123.1),
		),
	)
	var coverage *Coverage
	s.Init(
		WithCoverageCollect(func(cov *Coverage) {
			coverage = cov
		}),
		WithTesting(t),
		WithHttpDo(&dummyDo{
			status: http.StatusOK,
			body:   []byte(`{"foo": "bar"}`),
		}),
		WithVar("OK", 201)).
		Run()
	require.NotNil(t, coverage)
	fmt.Printf("coverage: %+v\n", coverage)
}

type dummyDo struct {
	status int
	body   []byte
}

func (d *dummyDo) Do(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: d.status,
		Body:       io.NopCloser(bytes.NewReader(d.body)),
	}, nil
}
