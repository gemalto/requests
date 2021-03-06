// Package clientserver is a utility for writing HTTP tests.
//
// A ClientServer embeds an httptest.Server and
// a requests.Requests client.  The client is preconfigured
// to talk to the server.
package clientserver

import (
	"github.com/gemalto/requests"
	"net/http"
	"net/http/httptest"
)

// New creates a new ClientServer.
//
// Panics if the option arguments cause an error.
func New(s *httptest.Server, options ...requests.Option) *ClientServer {
	if s == nil {
		s = httptest.NewServer(nil)
	}
	t := &ClientServer{
		Server: s,
		Requests: &requests.Requests{
			Doer: s.Client(),
		},
	}
	s.Config.Handler = t

	err := t.Apply(requests.URL(s.URL), optionsSlice(options), requests.Use(t.captureClientReqResp))
	if err != nil {
		panic(err)
	}

	return t
}

type optionsSlice []requests.Option

func (o optionsSlice) Apply(r *requests.Requests) error {
	return r.Apply(o...)
}

// A ClientServer is an http server and an http client.  The client is preconfigured
// to talk to the server.  Because it embeds a requests.Requeests, it support all the
// same methods for composing and sending HTTP requests, which are send to the embedded
// HTTP server.
//
// Should be closed at the end of the test.
type ClientServer struct {
	*httptest.Server
	*requests.Requests
	Handler http.Handler

	// These arguments are populated automatically during each
	// request.  Use Clear() to clear them between tests.

	// The last request handled by the server.
	LastSrvReq *http.Request

	// The last request sent by the client.
	LastClientReq *http.Request

	// The last response received by the client.
	LastClientResp *http.Response
}

// Close shuts down the embedded HTTP server.
func (t *ClientServer) Close() {
	t.Server.Close()
}

// Clear clears the attributes captured by the last request.
func (t *ClientServer) Clear() {
	t.LastClientReq = nil
	t.LastClientResp = nil
	t.LastSrvReq = nil
}

// ServerHTTP implements http.Handler.  ClientServer installs itself as the
// server's Handler so it can capture the request.  It then delegates to
// the Handler attribute.
func (t *ClientServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	t.LastSrvReq = req
	if t.Handler != nil {
		t.Handler.ServeHTTP(w, req)
	}
}

func (t *ClientServer) captureClientReqResp(next requests.Doer) requests.Doer {
	return requests.DoerFunc(func(req *http.Request) (*http.Response, error) {
		t.LastClientReq = req
		resp, err := next.Do(req)
		t.LastClientResp = resp
		return resp, err
	})
}

// Mux returns a ServeMux.  If the current Handler is a ServeMux, that
// is returned.  Otherwise, a new ServerMux is created and installed as
// the handler.
func (t *ClientServer) Mux() *http.ServeMux {
	if m, ok := t.Config.Handler.(*http.ServeMux); ok {
		return m
	}
	m := http.NewServeMux()
	t.Config.Handler = m
	return m
}

// HandlerFunc is a convience method for installing a HandlerFunc as the
// handler.
func (t *ClientServer) HandlerFunc(hf http.HandlerFunc) {
	t.Handler = hf
}
