package common

import "net/http"

type HttpDo interface {
	Do(req *http.Request) (*http.Response, error)
}
