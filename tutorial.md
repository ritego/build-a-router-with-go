# RiteGo - Build a Router with Go

    The full source code for this tutorial is available at [Build a Router with Go](https//github.com/ritego/build-a-router-with-go). 

## Start with Mux
Mux is the short and nice way to say (multiplexer)[https://en.wikipedia.org/wiki/Multiplexer]. In electronics, the primary source of the word, a multiplexer is a device that produce a single signal output based on several possible signal inputs. The output is a function of the input signal and some additional login. 

Interestingly, the concept of a multiplexer fits into requests (inputs) and responses (outputs) pattern of the HTTP protocol. 

In this tutorial, we are going to build a custom HTTP multiplexer (or better still a mux or a router). Our router, we would use router going forward, would match incoming URLs against a list of predefined patterns. The corresponding handler of the pattern that closely matches the URL is executed.

## Builtin Mux
Go's default router is `http.ServeMux`. THis implementation is exposed to developers through:

1. `DefaultServeMux`
This is the defaul router used when handlers are registered directly against `http` without a custom mux instantiated. For instance:
```go
    http.HandleFunc("/route1", func (w http.ResponseWriter, r *http.Request) { })
    http.HandleFunc("/route2", func (w http.ResponseWriter, r *http.Request) { })
    http.ListenAndServe(":7777", nil)
```
this is the same thing as 
```go
    router := http.DefaultServeMux
    router.HandleFunc("/route1", func (w http.ResponseWriter, r *http.Request) { })
    router.HandleFunc("/route2", func (w http.ResponseWriter, r *http.Request) { })
    http.ListenAndServe(":7777", router)
```

2. `NewServeMux`
API consumers can use `NewServeMux` to instantiate a custom router, modify it, and attach route handlers to it. The approach allows us to provide additional configuration to `ServeMux`.
```go
    router := http.NewServeMux()
    router.HandleFunc("/route1", func (w http.ResponseWriter, r *http.Request) { })
    router.HandleFunc("/route2", func (w http.ResponseWriter, r *http.Request) { })
    http.ListenAndServe(":7777", router)
```
## Third Party Muxes
(Gorilla Mux)[https://github.com/gorilla/mux] is a full featured router for Go. 

## Our Own Router
We would call our router just `router`. It would be simple with the following requirements:
- match routes bases on `method`, `host` and `path` only. Production routers are usually more robust with path matching covering URL queries, header values and schemes.
- routes definition would be plain, such as `user/assets`, No regex would be supported.
- our router must be able to work as a drop-in replacement for `http.ServeMux`. This basically means that it has to implement the (http.Handler) interface

## Setup Module
First thing first, setup your environment
1. Run the following in your root directory: `$ go mod init github.com/ritego/build-a-router-with-go` to generate the `go.mod` file
```mod
// go. mod
module github.com/ritego/build-a-router-with-go

go 1.16
```

## Utilities
Our router would need some utilities like custom errors and common functions. 

We would need to return these errors at different point in the future.
```go
var (
	ErrMethodNotAllowed = errors.New("method is not allowed for this router")
	ErrBadPath          = errors.New("every path definition must conform to [Method]:[Url]")
)
```

The following would also come in handy later. The first would enable us enforce valid HTTP methods, while the later would be used to sanitize and parse route paths during registration.
```go
func isValidMethod(method string) bool {
	for _, m := range []string{"GET", "POST", "PUT", "DELETE", "HEAD", "OPTIONS"} {
		if strings.EqualFold(m, method) {
			return true
		}
	}
	return false
}

func tokenize(path string) (string, string, string) {
	paths := strings.Split(path, ":")
	if len(paths) != 2 {
		panic(ErrBadPath)
	}

	pathMethod := paths[0]
	if !isValidMethod(pathMethod) {
		panic(ErrMethodNotAllowed)
	}

	pathUrl := paths[1]
	pathUrl = strings.TrimPrefix(pathUrl, "/")
	pathUrl = strings.TrimSuffix(pathUrl, "/")

	if pathUrl == "" {
		pathUrl = "/"
	}

	u, err := url.Parse(pathUrl)
	if err != nil {
		panic(err)
	}

	return pathMethod, u.Host, u.Path
}
```

## A Route 
We start by defining two containers: `Route` and `Router`. `Route` holds a single route definition together and `Router` is the registry for all our routes. Comparison and route matches would be carried oput by the `Router` when processing requests.
```go

type Route struct {
	path    string
	handler http.Handler
}

type Router struct {
	mu           *sync.Mutex
	routes       []Route
}
```

## The Router
Our `Router` needs has two important duties:
1. Maintain a registry for routes. To do these, we would implement two methods on the `Router` i.e `Handle` and `HandlerFunc`. The later is just a wrapper around the former to enable registering anonymous functions as a handler.

Note that we use a  mutual exclusion lock (sync.mutex) to synchronize changes on our `Router` across goroutines, in the case that `Route` registration is attempted by multiple `goroutines`. And we use our `tokenize` utility function to clean and unify paths.
```go
func (r *Router) Handle(path string, handler http.Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if handler == nil {
		panic(fmt.Sprintf("router: nill handler provided for path %v", path))
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
```

2. Match incoming request to the correct handler. To achieve this we would implement a `Handler` method that takes in (*http.Request) and return a single (http.Handler). This method is called by the `http.Server` when running for route resolution. 

```go
func (r *Router) ServeHTTP(rw http.ResponseWriter, rr *http.Request) {
	handler := r.match(rr)
	handler.ServeHTTP(rw, rr)
}

func (r *Router) match(rr *http.Request) http.Handler {
	incomingRoute := &Route{
		method: rr.Method,
		host:   rr.URL.Host,
		path:   rr.URL.Path,
	}

	for _, route := range r.routes {
		if route.method == incomingRoute.method && route.host == incomingRoute.host && route.path == incomingRoute.path {
			incomingRoute.handler = route.handler
			break
		}
	}

	if incomingRoute.handler == nil {
		return http.NotFoundHandler()
	}

	return incomingRoute.handler
}
```

Our router is pretty much ready at this point, but we need to provide a clean interface for consumer of our router.
```go
func New() *Router {
	return &Router{}
}
```

## Let's Consume out Router
Our brand new router can be used as by configuring the router and passing it to a server instance.

First, we instantiate a new router and setup handlers for it:
```go
var rr = router.New()

rr.HandleFunc("GET:/", func(rw http.ResponseWriter, r *http.Request) {
    rw.Write([]byte("Root - Hello World!"))
})

rr.HandleFunc("GET:/path-one", func(rw http.ResponseWriter, r *http.Request) {
    rw.Write([]byte("Path One - Hello World!"))
})

rr.HandleFunc("GET:/path-one/path-two", func(rw http.ResponseWriter, r *http.Request) {
    rw.Write([]byte("Path One - Hello World!"))
})

```

Then, we pass the router instance to our server for request handling:
```go
	srv := &http.Server{
		Handler:      rr,
		Addr:         ":7777",
	}

	log.Printf("Server running on: %s", addr)

	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
```

lastly, we run `go run main.go` to start accepting request. You can visit the defined routes (e.g http://127.0.0.1:7777/path-one) to the responses.

Just to recap, this is how route resolution is done in Go:
1. Once our (http.Server) is up and running, it listen for incoming request on the defined port.
2. For any incoming request, the (http.Server) would invoke invoke the `Handler.ServeHttp` of the register `Handler`. Which in this case is our router. Remember that our router also implements the (Handler) interface. 
3. Our handler, in turn, compares the request with registered `Routes`, select the first matching (http.Handler) from the entries and invokes the corresponding `Handler.ServeHttp` of the handler. Also, remember that all route handlers implements the (Handler) interface.

This means that both our router and all registered route handler must satisfy the (Handler) interface. The `Handler.ServeHttp` is very important for request resolution, in fact, requests are resolved through cascaded calls to`Handler.ServeHttp`

## Conclusion
The full source code for this tutorial is available at [Build a Router with Go](https://github.com/ritego/build-a-router-with-go).
