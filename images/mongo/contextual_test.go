package mongo

import (
	"bytes"
	"github.com/go-andiamo/marrow"
	"github.com/go-andiamo/marrow/common"
	"github.com/go-andiamo/marrow/coverage"
	"github.com/go-andiamo/marrow/with"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
	mdb "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"io"
	"net/http"
	"testing"
)

func TestResolvablesAndBeforeAfters(t *testing.T) {
	do := &dummyDo{
		status: http.StatusOK,
		body:   []byte(`{"foo":"bar"}`),
	}
	opts := Options{
		CreateIndices: IndexOptions{
			"my-db": map[string][]mdb.IndexModel{
				"my-collection": {
					{
						Keys:    bson.D{{Key: "email", Value: 1}},
						Options: options.Index().SetName("uniq_email").SetUnique(true),
					},
				},
			},
		},
	}
	endpoint := marrow.Endpoint("/api", "",
		marrow.Method("GET", "").AssertOK().
			AssertEqual(4, DatabasesCount()).
			AssertEqual(1, CollectionsCount("my-db")).
			AssertEqual(2, IndicesCount("my-db", "my-collection")).
			AssertEqual(0, DocumentsCount("my-db", "my-collection")).
			Capture(InsertDocument(marrow.After, "new-db", "new-coll", marrow.JSON{"email": "bilbo@example.com"})),
		marrow.Method("GET", "again").AssertOK().
			SetVar(marrow.Before, "find-email", "bilbo@example.com").
			SetVar(marrow.Before, "doc", FindOne("new-db", "new-coll", marrow.JSON{"email": marrow.Var("find-email")}, nil)).
			AssertEqual(5, DatabasesCount()).
			AssertEqual(1, CollectionsCount("new-db")).
			AssertEqual(1, IndicesCount("new-db", "new-coll")).
			AssertEqual(1, DocumentsCount("new-db", "new-coll")).
			AssertEqual("bilbo@example.com", marrow.JsonPath(marrow.Var("doc"), "email")).
			AssertEqual(1, marrow.JsonPath(Find("new-db", "new-coll", nil, nil), marrow.LEN)).
			AssertEqual("bilbo@example.com", marrow.JsonTraverse(Find("new-db", "new-coll", nil, nil), 0, "email")).
			AssertEqual(1, marrow.JsonPath(Query("new-db", `{ "find": "new-coll", "filter": { "email": { "$eq": "bilbo@example.com" } } }`), "LEN")).
			AssertEqual("bilbo@example.com", marrow.JsonTraverse(Query("new-db", `{ "find": "new-coll", "filter": { "email": { "$eq": "bilbo@example.com" } } }`), 0, "email")),
		marrow.Method("GET", "again").AssertOK().
			Capture(ClearCollection(marrow.Before, "new-db", "new-coll")).
			Capture(DeleteDocuments(marrow.Before, "new-db", "new-coll", nil)).
			AssertEqual(0, DocumentsCount("new-db", "new-coll")),
	)
	var cov *coverage.Coverage
	s := marrow.Suite(endpoint).Init(
		With(opts),
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
