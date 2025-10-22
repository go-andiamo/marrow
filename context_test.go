package marrow

import "net/http"

func newContext(vars map[Var]any) *context {
	result := &context{
		coverage:  newCoverage(),
		vars:      make(map[Var]any),
		cookieJar: make(map[string]*http.Cookie),
	}
	for k, v := range vars {
		result.vars[k] = v
	}
	return result
}
