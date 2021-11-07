# RiteGo - Build a Router with Go

    The full source code for this tutorial is available at [Build a Router with Go](https//github.com/ritego/build-a-router-with-go). 

## Start with Mux
Mux is the short and nice way to say [multiplexer](https://en.wikipedia.org/wiki/Multiplexer). In electronics, the primary source of the word, a multiplexer is a device that produce a single signal output based on several possible signal inputs. The output is a function of the input signal and some additional logic. 

Interestingly, the concept of a multiplexer fits into requests (inputs) and responses (outputs) pattern of the HTTP protocol. 

In this tutorial, we are going to build a custom HTTP multiplexer (or better still a mux or a router). Our router would match incoming URLs against a list of predefined patterns. The corresponding handler of the pattern that closely matches the URL is invoked to handle the incoming request.

## Builtin Mux
Go's default router is [http.ServeMux](https://pkg.go.dev/net/http#ServeMux). The implementation is exposed to developers through:

1. [DefaultServeMux](https://pkg.go.dev/net/http#DefaultServeMux) - This is the default router used when handlers are registered directly against `http` without a custom mux instantiated. For instance:
	```go
	http.HandleFunc("/route1", func (w http.ResponseWriter, r *http.Request) { })
	http.HandleFunc("/route2", func (w http.ResponseWriter, r *http.Request) { })
	http.ListenAndServe(":7777", nil)
	```

	This is the same thing as 
	```go
	router := http.DefaultServeMux
	router.HandleFunc("/route1", func (w http.ResponseWriter, r *http.Request) { })
	router.HandleFunc("/route2", func (w http.ResponseWriter, r *http.Request) { })
	http.ListenAndServe(":7777", router)
	```

2. [NewServeMux](https://pkg.go.dev/net/http#NewServeMux) - This is used to instantiate a custom router, modify it, and attach route handlers to it. The approach allows us to provide additional configuration to `ServeMux`.
	```go
	router := http.NewServeMux()
	// perform some magic with router here
	router.HandleFunc("/route1", func (w http.ResponseWriter, r *http.Request) { })
	router.HandleFunc("/route2", func (w http.ResponseWriter, r *http.Request) { })
	http.ListenAndServe(":7777", router)
	```

## Our Own Router
Now that we have a rough idea of how the default router works, lets take a short at building our own.

We would call our router just `router`. It would be simple with the following requirements:
- match routes bases on `method`, `host` and `path` only. Production routers are usually more robust with path matching capability covering URL queries, header values and schemes.
- routes definition would be plain and simple (such as `GET:/user/assets`), no regex or variables would be supported.
- our router must be able to work as a drop-in replacement for `http.ServeMux`. This basically means that it has to implement the [http.Handler](https://pkg.go.dev/net/http#Handler) interface.

	```go
	type Handler interface {
		ServeHTTP(ResponseWriter, *Request)
	}
	```
## Setup Module
First thing first, we setup our environment. Run the following in your root directory: ` go mod init github.com/[username]/[name]`. This would setup a package module and generate both `go.mod` and `go.sum` files.


## File Structure
Our source source code would be laid out in the following way
```
- go.mod
- go.sum
- main.go // would tie everything together
- router // our mux/router would be in this sub package
- - index.go // holds utility functions
- - route.go // holds definition for a route
- - router.go // holds implementation of our router
```

## Utilities
Our router would need some utilities like custom errors and common functions. 

We would need to return these errors at different points in the future.
```go
// router/index.go

var (
	ErrMethodNotAllowed = errors.New("method is not allowed for this router")
	ErrBadPath          = errors.New("every path definition must conform to [Method]:[Url]")
	ErrNilHandler       = errors.New("nill handler provided")
)
```

The following would also come in handy later. The first would enable us enforce valid HTTP methods, while the later would be used to sanitize and parse route paths during registration.
```go
// router/index.go

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
We start by defining two containers: `Route` and `Router`. `Route` holds a single route definition together and `Router` is the registry for all our routes. Comparison and route matches would be carried out by the `Router` when processing requests.
```go
// router/route.go

type Route struct {
	method  string
	host    string
	path    string
	handler http.Handler
}
```

```go
// router/router.go

type Router struct {
	mu     sync.Mutex
	routes []Route
}
```

## The Router
Our `Router` has two important duties:
1. Maintain a registry for routes. To do these, we would implement two methods on the `Router` i.e `Handle` and `HandleFunc`. The later is a wrapper around the former to enable registering anonymous functions as a handler.

Note that we use a mutual exclusion lock (sync.Mutex) to synchronize changes on our `Router` across goroutines, in the case that `Route` registration is attempted by multiple `goroutines` at the same time. And we use our `tokenize` utility function to clean and unify paths.
```go
// router/router.go

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
```

2. Match incoming request to the correct handler. To achieve this we would implement a `Handler` method that takes in [http.Request](https://pkg.go.dev/net/http#Request) and return a single [http.Handler](https://pkg.go.dev/net/http#Handler). The `ServeHttp` method of this `Handler` is called by the [http.Server](https://pkg.go.dev/net/http#Server) when resolving requests. 

```go
// router/router.go

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
```

Our router is pretty much ready at this point, but we need to provide a clean interface for consumers.
```go
// router/index.go

func New() *Router {
	return &Router{}
}
```

## Let's Consume out Router
Our brand new router can be used by configuring the router and passing it to a server instance.

First, we instantiate a new router and setup handlers for it:
```go
// main.go

var rr = router.New()

rr.HandleFunc("GET:/", func(rw http.ResponseWriter, r *http.Request) {
	rw.Write([]byte("Root - Hello World!"))
})

rr.HandleFunc("GET:/path-one", func(rw http.ResponseWriter, r *http.Request) {
	rw.Write([]byte("Path One - Hello World!"))
})

rr.HandleFunc("GET:/path-one/path-two", func(rw http.ResponseWriter, r *http.Request) {
	rw.Write([]byte("Path Two - Hello World!"))
})
```

Then, we pass the router instance to our server for request handling:
```go
srv := &http.Server{
	Handler:      rr,
	Addr:         addr,
	WriteTimeout: viper.GetDuration("SERVER_WRITE_TIMEOUT"),
	ReadTimeout:  viper.GetDuration("SERVER_READ_TIMEOUT"),
}

log.Printf("Server running on: %s", addr)

if err := srv.ListenAndServe(); err != nil {
	log.Fatal(err)
}
```

lastly, we run `go run main.go` to start accepting request. You can visit the defined routes (e.g http://127.0.0.1:7777/path-one) to get the responses.

## How Does Go Resolve Request
Just to recap, this is how route resolution is done in Go:
1. Once our [http.Server](https://pkg.go.dev/net/http#Server) is up and running, it listens to incoming request on the defined port.
2. For any incoming request, the [http.Server](https://pkg.go.dev/net/http#Server) would invoke invoke the `ServeHttp` method of the registered `Handler`. Which in this case is our router. Remember that our router implements the [Handler](https://pkg.go.dev/net/http#Handler) interface. 
3. Our handler compares the request with registered `Routes`, selects the first matching [http.Handler](https://pkg.go.dev/net/http#Handler) from the entries and invokes the corresponding `ServeHttp` method  of the `Handler`. Also, remember that all route handlers implements the [Handler](https://pkg.go.dev/net/http#Handler) interface.

This means that both our router and all registered route handler must satisfy the [Handler](https://pkg.go.dev/net/http#Handler) interface. The `Handler.ServeHttp` is very important for request resolution. In fact, requests are resolved through cascaded calls to `Handler.ServeHttp`

## Conclusion
The full source code for this tutorial is available at [Build a Router with Go](https://github.com/ritego/build-a-router-with-go).
