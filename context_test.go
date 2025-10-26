package marrow

import (
	"github.com/go-andiamo/marrow/coverage"
	"net/http"
)

func newContext(vars map[Var]any) *context {
	result := &context{
		coverage:  coverage.NewNullCoverage(),
		vars:      make(map[Var]any),
		cookieJar: make(map[string]*http.Cookie),
	}
	for k, v := range vars {
		result.vars[k] = v
	}
	return result
}
