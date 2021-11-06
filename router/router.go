package router

import (
	"fmt"
	"net/http"
	"sync"
)

type Router struct {
	mu     sync.Mutex
	routes []Route
}

func (r *Router) Handle(path string, handler http.Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if handler == nil {
		panic(ErrNilHandler)
	}

	method, host, path := tokenize(path)

	r.routes = append(r.routes, Route{method, host, path, handler})
}

func (r *Router) HandleFunc(path string, handler func(rw http.ResponseWriter, rr *http.Request)) {
	if handler == nil {
		panic("router: nill handler provided")
	}
	r.Handle(path, http.HandlerFunc(handler))
}

func (r *Router) ServeHTTP(rw http.ResponseWriter, rr *http.Request) {
	handler := r.match(rr)
	handler.ServeHTTP(rw, rr)
}

func (r *Router) match(rr *http.Request) http.Handler {
	method, host, path := tokenize(rr.Method + ":" + rr.URL.Path)

	var handler http.Handler
	for _, route := range r.routes {
		fmt.Println(route)
		if route.method == method && route.host == host && route.path == path {
			handler = route.handler
			break
		}
	}

	if handler == nil {
		return http.NotFoundHandler()
	}

	return handler
}
