package weavebox

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"

	"github.com/bmizerany/pat"
	"github.com/gorilla/handlers"
	"golang.org/x/net/context"
)

var defaultErrHandler = func(w http.ResponseWriter, r *http.Request, err error) {
	log.Println("using weavebox default errorHandler, did you now you can use a custom one?")
	log.Fatal(err)
}

type Weavebox struct {
	router          *router
	NotFoundHandler http.Handler
	output          io.Writer
}

// New returns a new weavebox object
func New() *Weavebox {
	return &Weavebox{
		router: &router{
			PatternServeMux: pat.New()},
	}
}

func (w *Weavebox) init() http.Handler {
	if w.router.errorHandler == nil {
		w.router.errorHandler = defaultErrHandler
	}
	if w.output == nil {
		w.output = os.Stdout
	}

	return handlers.LoggingHandler(w.output, w.router)
}

// Serve serves the application on the given port
func (w *Weavebox) Serve(port int) error {
	h := w.init()
	log.Printf("listening on 0.0.0.0:%d", port)
	return http.ListenAndServe(fmt.Sprintf(":%d", port), h)
}

// ServeTLS server the application with TLS encription on the given port
func (w *Weavebox) ServeTLS(port int, certFile, keyFile string) error {
	h := w.init()
	portStr := fmt.Sprintf(":%d", port)
	log.Printf("listening TLS on 0.0.0.0:%d", port)
	return http.ListenAndServeTLS(portStr, certFile, keyFile, h)
}

// Handler is an opinionated / idiom how weavebox thinks a request handler
// should look like. It carries a context, responseWriter, request and
// returns an error. Errors returned by Handler can be catched by setting a
// custom errorHandler, see SetErrorHandler more information.
type Handler func(ctx *Context, w http.ResponseWriter, r *http.Request) error

// Get registers a route prefix and will invoke the Handler when the route
// matches the prefix and the request METHOD is GET
func (w *Weavebox) Get(route string, h Handler) {
	w.router.add("GET", route, h)
}

// Post registers a route prefix and will invoke the Handler when the route
// matches the prefix and the request METHOD is POST
func (w *Weavebox) Post(route string, h Handler) {
	w.router.add("POST", route, h)
}

// Put registers a route prefix and will invoke the Handler when the route
// matches the prefix and the request METHOD is PUT
func (w *Weavebox) Put(route string, h Handler) {
	w.router.add("PUT", route, h)
}

// Delete registers a route prefix and will invoke the Handler when the route
// matches the prefix and the request METHOD is DELETE
func (w *Weavebox) Delete(route string, h Handler) {
	w.router.add("DELETE", route, h)
}

// Static registers the prefix as a static fileserver for dir
func (w *Weavebox) Static(prefix string, dir string) {
	h := http.StripPrefix(prefix, http.FileServer(http.Dir(dir)))
	w.router.Add("GET", prefix, h)
}

// Subrouter returns a new Weavebox object that acts as a subrouter.
// Subrouters will inherit the parents allready defined middleware.
// Subrouters can have there own errorHandler and middleware.
// Overiding middleware can be done by calling Middleware on the subrouter
// instead of calling Register. See the example for detailed information.
func (w *Weavebox) Subrouter(prefix string) *Weavebox {
	return &Weavebox{
		router: &router{
			PatternServeMux: w.router.PatternServeMux,
			handlers:        w.router.handlers,
			prefix:          prefix,
			errorHandler:    w.router.errorHandler,
		},
	}
}

// SetOutput will write a default appache log the given writer
func (weav *Weavebox) SetOutput(w io.Writer) {
	weav.output = w
}

// SetErrorHandler will register a errHandleFunc to the router and will
// handle all errors return by a weave Handler.
func (w *Weavebox) SetErrorHandler(fn errHandlerFunc) {
	w.router.errorHandler = fn
}

// Middleware accepts a chain of weavebox Handlers that are invoked in order,
// before invoking the final handler, that is set by calling (Get, Put, Post, Delete).
// Middleware can be called on subrouters to override the parents middleware.
// To append middleware use Register instead.
func (w *Weavebox) Middleware(handlers ...Handler) {
	w.router.handlers = handlers
}

// Register appends a single Handler to the middleware. Register can be called
// on subrouters to add different middleware for each subrouter.
func (w *Weavebox) Register(h Handler) {
	w.router.handlers = append(w.router.handlers, h)
}

type router struct {
	*pat.PatternServeMux
	prefix       string
	handlers     []Handler
	errorHandler errHandlerFunc
}

func (r *router) add(method, route string, h Handler) {
	r.Add(method, path.Join(r.prefix, route), r.makeHttpHandler(h))
}

type errHandlerFunc func(w http.ResponseWriter, r *http.Request, err error)

func (router *router) makeHttpHandler(h Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := &Context{context.Background(), r.URL.Query()}
		for _, handler := range router.handlers {
			if err := handler(ctx, w, r); err != nil {
				router.errorHandler(w, r, err)
				return
			}
		}
		if err := h(ctx, w, r); err != nil {
			router.errorHandler(w, r, err)
			return
		}
	}
}

// Context is required in each weavebox handler, and can be used to pass
// information between requests.
type Context struct {
	// Context is idiomatic way for passing information between requests.
	// More information about context.Context can be found here:
	// https://godoc.org/golang.org/x/net/context
	Context context.Context

	// Vars carries request URL parameters that are passed in route prefixes.
	// ex. /order/:id => Vars.Get(":id")
	Vars url.Values
}

// JSON is a helper function for writing a JSON encoded representation of v
// to the ResponseWriter
func JSON(w http.ResponseWriter, code int, v interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	return json.NewEncoder(w).Encode(v)
}

// Text is a helper function for writing plain text.
func Text(w http.ResponseWriter, code int, str string) error {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(code)
	w.Write([]byte(str))
	return nil
}
