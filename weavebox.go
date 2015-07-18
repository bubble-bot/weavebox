package weavebox

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
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

// New returns weavebox object with a default mux router attached
func New() *Weavebox {
	return &Weavebox{
		router: &router{Router: mux.NewRouter()},
	}
}

func (w *Weavebox) init() http.Handler {
	w.router.NotFoundHandler = w.NotFoundHandler

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

// ServeTLS server the application with TLS encription
func (w *Weavebox) ServeTLS(port int, certFile, keyFile string) error {
	h := w.init()
	portStr := fmt.Sprintf(":%d", port)
	log.Printf("listening TLS on 0.0.0.0:%d", port)
	return http.ListenAndServeTLS(portStr, certFile, keyFile, h)
}

// Handler is an opinionated / idiom http handler how weavebox thinks a request
// handler should look like. It carries a context, responseWriter, request and
// returns an error. Errors returned by Handler can be catched by setting a
// custom errorHandler, see SetErrorHandler for details
type Handler func(ctx *Context, w http.ResponseWriter, r *http.Request) error

// Get registers a route prefix and will invoke the Handler when the route
// matches the prefix and the request METHOD is GET
func (w *Weavebox) Get(route string, h Handler) {
	w.router.add(route, "GET", h)
}

// Post registers a route prefix and will invoke the Handler when the route
// matches the prefix and the request METHOD is POST
func (w *Weavebox) Post(route string, h Handler) {
	w.router.add(route, "POST", h)
}

// Put registers a route prefix and will invoke the Handler when the route
// matches the prefix and the request METHOD is PUT
func (w *Weavebox) Put(route string, h Handler) {
	w.router.add(route, "PUT", h)
}

// Delete registers a route prefix and will invoke the Handler when the route
// matches the prefix and the request METHOD is DELETE
func (w *Weavebox) Delete(route string, h Handler) {
	w.router.add(route, "DELETE", h)
}

// Static registers the prefix as a static fileserver for dir
func (w *Weavebox) Static(prefix string, dir string) {
	w.router.PathPrefix(prefix).Handler(http.FileServer(http.Dir(dir)))
}

// Subrouter returns a new Weavebox object that acts as a subrouter.
// For each subrouter a new errorHandler can be set. If no errorHandler
// is set for the subset of routes the parent errorHandler wil be invoked.
func (w *Weavebox) Subrouter(route string) *Weavebox {
	r := w.router.PathPrefix(route).Subrouter()
	return &Weavebox{
		router: &router{
			Router:       r,
			errorHandler: w.router.errorHandler,
		},
	}
}

// SetOutput will write a default appache log the given writer
func (weav *Weavebox) SetOutput(w io.Writer) {
	weav.output = w
}

// SetErrorHandler will handle all errors returned from a Handler
func (w *Weavebox) SetErrorHandler(fn errHandlerFunc) {
	w.router.errorHandler = fn
}

// Middleware accepts a chain of weavebox Handlers that are invoked in order
// before invoking the final handler set by calling (Get, Put, Post, Delete)
func (w *Weavebox) Middleware(handlers ...Handler) {
	w.router.handlers = handlers
}

type router struct {
	*mux.Router
	handlers     []Handler
	errorHandler errHandlerFunc
}

func (r *router) add(route, method string, h Handler) {
	f := r.makeHttpHandler(h)
	r.Path(route).Methods(method).Handler(f)
}

type errHandlerFunc func(w http.ResponseWriter, r *http.Request, err error)

func (router *router) makeHttpHandler(h Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := &Context{context.Background(), mux.Vars(r)}
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

	// Vars carries request url parameters passes in route prefixes
	// ex. /order/{id} => get the id by calling Vars["id"]
	Vars map[string]string
}

// JSON is a helper function for writing a JSON encoded representation of v
// to the ResonseWriter
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
