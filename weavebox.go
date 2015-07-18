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

type weavebox struct {
	router          *router
	NotFoundHandler http.Handler
	output          io.Writer
}

// New returns weavebox object with a default mux router attached
func New() *weavebox {
	return &weavebox{
		router: &router{Router: mux.NewRouter()},
	}
}

func (w *weavebox) init() http.Handler {
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
func (w *weavebox) Serve(port int) error {
	h := w.init()
	log.Printf("listening on 0.0.0.0:%d", port)
	return http.ListenAndServe(fmt.Sprintf(":%d", port), h)
}

// ServeTLS server the application with TLS encription
func (w *weavebox) ServeTLS(port int, certFile, keyFile string) error {
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
// matches the prefix
func (w *weavebox) Get(route string, h Handler) {
	w.router.add(route, "GET", h)
}

func (w *weavebox) Post(route string, h Handler) {
	w.router.add(route, "POST", h)
}

func (w *weavebox) Put(route string, h Handler) {
	w.router.add(route, "PUT", h)
}

func (w *weavebox) Delete(route string, h Handler) {
	w.router.add(route, "DELETE", h)
}

// Static registers the prefix as a static fileserver for dir
func (w *weavebox) Static(prefix string, dir string) {
	w.router.PathPrefix(prefix).Handler(http.FileServer(http.Dir(dir)))
}

// Subrouter will inherit the errHandleFunc
func (w *weavebox) Subrouter(route string) *weavebox {
	r := w.router.PathPrefix(route).Subrouter()
	return &weavebox{
		router: &router{
			Router:       r,
			errorHandler: w.router.errorHandler,
		},
	}
}

// SetOutput will write a default appache log the given writer
func (weav *weavebox) SetOutput(w io.Writer) {
	weav.output = w
}

// SetErrorHandler will handle all errors returned from a Handler
func (w *weavebox) SetErrorHandler(fn errHandlerFunc) {
	w.router.errorHandler = fn
}

func (w *weavebox) Middleware(handlers ...Handler) {
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

type Context struct {
	Context context.Context
	Vars    map[string]string
}

func JSON(w http.ResponseWriter, code int, v interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	return json.NewEncoder(w).Encode(v)
}

func Text(w http.ResponseWriter, code int, str string) error {
	w.Header().Set("Content-Type", "text//plain")
	w.WriteHeader(code)
	w.Write([]byte(str))
	return nil
}
