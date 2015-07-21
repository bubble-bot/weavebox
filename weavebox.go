package weavebox

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/httprouter"
	"golang.org/x/net/context"
)

// weavebox is opinion based minimalistic web framework for making fast and
// powerfull web application in the Go programming language. It is backed by
// the fastest and most optimized request router available. Weavebox also
// provides a gracefull webserver that can serve TLS encripted requests aswell.

var defaultErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

type Weavebox struct {
	// ErrorHandler is invoked whenever a Handler returns an error
	ErrorHandler ErrorHandlerFunc

	// NotFoundHandler is invoked whenever the router could not match a route
	// against the request url
	NotFoundHandler http.Handler

	// Output writes the access-log and debug parameters
	Output io.Writer

	// EnableLog lets you turn of the default access-log
	EnableLog  bool
	router     *httprouter.Router
	middleware []Handler
	prefix     string
}

// New returns a new Weavebox object
func New() *Weavebox {
	return &Weavebox{
		router:       httprouter.New(),
		Output:       os.Stderr,
		ErrorHandler: defaultErrorHandler,
		EnableLog:    true,
	}
}

// Serve serves the application on the given port
func (w *Weavebox) Serve(port int) error {
	w.init()
	portStr := fmt.Sprintf(":%d", port)
	fmt.Fprintf(w.Output, "app listening on 0.0.0.0:%d\n", port)
	return ListenAndServe(portStr, w)
}

// ServeTLS servers the application one the given port with TLS encription.
func (w *Weavebox) ServeTLS(port int, keyFile, certFile string) error {
	w.init()
	portStr := fmt.Sprintf(":%d", port)
	fmt.Fprintf(w.Output, "app listening on 0.0.0.0:%d\n", port)
	return ListenAndServeTLS(portStr, w, keyFile, certFile)
}

// Get registers a route prefix and will invoke the Handler when the route
// matches the prefix and the request METHOD is GET
func (w *Weavebox) Get(route string, h Handler) {
	w.add("GET", route, h)
}

// Post registers a route prefix and will invoke the Handler when the route
// matches the prefix and the request METHOD is POST
func (w *Weavebox) Post(route string, h Handler) {
	w.add("POST", route, h)
}

// Put registers a route prefix and will invoke the Handler when the route
// matches the prefix and the request METHOD is PUT
func (w *Weavebox) Put(route string, h Handler) {
	w.add("PUT", route, h)
}

// Delete registers a route prefix and will invoke the Handler when the route
// matches the prefix and the request METHOD is DELETE
func (w *Weavebox) Delete(route string, h Handler) {
	w.add("DELETE", route, h)
}

// Static registers the prefix to the router and start to act as a fileserver
// ex. "/public", "./assets"
func (w *Weavebox) Static(prefix, dir string) {
	w.router.ServeFiles(path.Join(prefix, "*filepath"), http.Dir(dir))
}

// Use appends a Handler to the box middleware. Different middleware can be set
// for each subrouter (Box).
func (w *Weavebox) Use(handlers ...Handler) {
	for _, h := range handlers {
		w.middleware = append(w.middleware, h)
	}
}

// Subrouter returns a new Box that will inherit all of its parents middleware.
// you can reset the middleware registered to the box by calling Reset()
func (w *Weavebox) Subrouter(prefix string) *Box {
	b := &Box{*w}
	b.Weavebox.prefix += prefix
	return b
}

// ServeHTTP
func (w *Weavebox) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	if w.EnableLog {
		start := time.Now()
		logger := &responseLogger{w: rw}
		w.router.ServeHTTP(logger, r)
		w.writeLog(r, start, logger.Status(), logger.Size())
		// saves an allocation by seperating the whole logger if log is disabled
	} else {
		w.router.ServeHTTP(rw, r)
	}
}

// Box act as a subrouter and wil inherit all of its parents middleware
type Box struct {
	Weavebox
}

// Reset clears all middleware
func (b *Box) Reset() *Box {
	b.Weavebox.middleware = nil
	return b
}

func (w *Weavebox) init() {
	if w.NotFoundHandler != nil {
		w.router.NotFound = w.NotFoundHandler
	}
}

func (w *Weavebox) add(method, route string, h Handler) {
	path := path.Join(w.prefix, route)
	w.router.Handle(method, path, w.makeHTTPRouterHandle(h))
}

func (w *Weavebox) makeHTTPRouterHandle(h Handler) httprouter.Handle {
	return func(rw http.ResponseWriter, r *http.Request, params httprouter.Params) {
		ctx := &Context{
			Context: context.Background(),
			Vars:    params,
		}
		for _, handler := range w.middleware {
			if err := handler(ctx, rw, r); err != nil {
				w.ErrorHandler(rw, r, err)
				return
			}
		}

		if err := h(ctx, rw, r); err != nil {
			w.ErrorHandler(rw, r, err)
			return
		}
	}
}

func (w *Weavebox) writeLog(r *http.Request, start time.Time, status, size int) {
	host, _, _ := net.SplitHostPort(r.Host)
	fmt.Fprintf(w.Output, "%s - [%s] %s %s %s %d %d %d\n",
		host,
		start.Format("02/Jan/2006:15:04:05 -0700"),
		r.Method,
		r.RequestURI,
		r.Proto,
		status,
		size,
		time.Now().Sub(start),
	)
}

// Handler is a opinion / idiom of how weavebox thinks a request handler should
// look like. It requires a Context, ResponseWriter, Request and returns an error
type Handler func(ctx *Context, w http.ResponseWriter, r *http.Request) error

// ErrorHandlerFunc is invoked when a Handler return an error and can be used
// to centralize error handling.
type ErrorHandlerFunc func(w http.ResponseWriter, r *http.Request, err error)

// Context is required in each weavebox Handler and can be used to pass information
// between requests.
type Context struct {
	// Context is a idiomatic way to pass information between requests.
	// More information about context.Context can be found here:
	// https://godoc.org/golang.org/x/net/context
	Context context.Context

	// Vars carries the named request URL parameters that are passed in the route
	// prefix. To get a parameter by name: Vars.GetByName(<param>)
	Vars httprouter.Params
}

// JSON is a helper function for writing a JSON encoded representation of v to
// the ResponseWriter.
func JSON(w http.ResponseWriter, code int, v interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	return json.NewEncoder(w).Encode(v)
}

// Text is a helper function for writing a text/plain string to the ResponseWriter
func Text(w http.ResponseWriter, code int, text string) error {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(code)
	w.Write([]byte(text))
	return nil
}

type responseLogger struct {
	w      http.ResponseWriter
	status int
	size   int
}

func (l *responseLogger) Write(p []byte) (int, error) {
	if l.status == 0 {
		l.status = http.StatusOK
	}
	size, err := l.w.Write(p)
	l.size += size
	return size, err
}

func (l *responseLogger) Header() http.Header {
	return l.w.Header()
}

func (l *responseLogger) WriteHeader(code int) {
	l.w.WriteHeader(code)
	l.status = code
}

func (l *responseLogger) Status() int {
	return l.status
}

func (l *responseLogger) Size() int {
	return l.size
}
