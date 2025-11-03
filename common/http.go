package common

import "net/http"

// HttpDo is the interface that describes how to make http calls
//
// http.DefaultClient is generally used, but this can be mocked by implementing this interface
type HttpDo interface {
	Do(req *http.Request) (*http.Response, error)
}
