package weavebox

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/julienschmidt/httprouter"
	"golang.org/x/net/context"
)

// Package weavebox is opinion based minimalistic web framework for making fast and
// powerfull web application in the Go programming language. It is backed by
// the fastest and most optimized request router available. Weavebox also
// provides a gracefull webserver that can serve TLS encripted requests aswell.

var defaultErrorHandler = func(ctx *Context, err error) {
	http.Error(ctx.Response(), err.Error(), http.StatusInternalServerError)
}

// Weavebox first class object that is created by calling New()
type Weavebox struct {
	// ErrorHandler is invoked whenever a Handler returns an error
	ErrorHandler ErrorHandlerFunc

	// Output writes the access-log and debug parameters
	Output io.Writer

	// EnableLog lets you turn of the default access-log
	EnableLog bool

	// HTTP2 enables the HTTP2 protocol on the server. HTTP2 wil be default proto
	// in the future. Currently browsers only supports HTTP/2 over encrypted TLS.
	HTTP2 bool

	templateEngine Renderer
	router         *httprouter.Router
	middleware     []Handler
	prefix         string
	context        context.Context
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
	srv := newServer(fmt.Sprintf(":%d", port), w, w.HTTP2)
	return w.serve(srv)
}

// ServeTLS servers the application one the given port with TLS encription.
// ServeTLS uses the HTTP2 protocol by default
func (w *Weavebox) ServeTLS(port int, certFile, keyFile string) error {
	srv := newServer(fmt.Sprintf(":%d", port), w, true)
	return w.serve(srv, certFile, keyFile)
}

// ServeCustom serves the application with custom server configuration.
func (w *Weavebox) ServeCustom(s *http.Server) error {
	return w.serve(s)
}

// ServeCustomTLS serves the application with TLS encription and custom server configuration.
func (w *Weavebox) ServeCustomTLS(s *http.Server, certFile, keyFile string) error {
	return w.serve(s, certFile, keyFile)
}

func (w *Weavebox) serve(s *http.Server, files ...string) error {
	srv := &server{
		Server: s,
		quit:   make(chan struct{}, 1),
		fquit:  make(chan struct{}, 1),
	}
	if len(files) == 0 {
		fmt.Fprintf(w.Output, "app listening on 0.0.0.0:%s\n", s.Addr)
		return srv.ListenAndServe()
	}
	if len(files) == 2 {
		fmt.Fprintf(w.Output, "app listening TLS on 0.0.0.0:%s\n", s.Addr)
		return srv.ListenAndServeTLS(files[0], files[1])
	}
	return errors.New("invalid server configuration")
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
// 	app.Static("/public", "./assets")
func (w *Weavebox) Static(prefix, dir string) {
	w.router.ServeFiles(path.Join(prefix, "*filepath"), http.Dir(dir))
}

// BindContext lets you provide a context that will live a full http roundtrip
// BindContext is mostly used in a func main() to provide init variables that
// may be created only once, like a database connection. If BindContext is not
// called, weavebox will use a context.Background()
func (w *Weavebox) BindContext(ctx context.Context) {
	w.context = ctx
}

// Use appends a Handler to the box middleware. Different middleware can be set
// for each subrouter (Box).
func (w *Weavebox) Use(handlers ...Handler) {
	for _, h := range handlers {
		w.middleware = append(w.middleware, h)
	}
}

// Box returns a new Box that will inherit all of its parents middleware.
// you can reset the middleware registered to the box by calling Reset()
func (w *Weavebox) Box(prefix string) *Box {
	b := &Box{*w}
	b.Weavebox.prefix += prefix
	return b
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

// SetTemplateEngine allows the use of any template engine out there, if it
// satisfies the Renderer interface
func (w *Weavebox) SetTemplateEngine(t Renderer) {
	w.templateEngine = t
}

// SetNotFoundHandler sets a custom notFoundHandler that is invoked whenever the
// router could not match a route against the request url.
func (w *Weavebox) SetNotFoundHandler(h http.Handler) {
	w.router.NotFound = h
}

// ServeHTTP satisfies the http.Handler interface
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

func (w *Weavebox) add(method, route string, h Handler) {
	path := path.Join(w.prefix, route)
	w.router.Handle(method, path, w.makeHTTPRouterHandle(h))
}

func (w *Weavebox) makeHTTPRouterHandle(h Handler) httprouter.Handle {
	return func(rw http.ResponseWriter, r *http.Request, params httprouter.Params) {
		if w.context == nil {
			w.context = context.Background()
		}
		ctx := &Context{
			Context:  w.context,
			vars:     params,
			response: rw,
			request:  r,
			weavebox: w,
		}
		for _, handler := range w.middleware {
			if err := handler(ctx); err != nil {
				w.ErrorHandler(ctx, err)
				return
			}
		}
		if err := h(ctx); err != nil {
			w.ErrorHandler(ctx, err)
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

// Handler is a weavebox idiom for handling http.Requests
type Handler func(ctx *Context) error

// ErrorHandlerFunc is invoked when a Handler returns an error, and can be used
// to centralize error handling.
type ErrorHandlerFunc func(ctx *Context, err error)

// Context is required in each weavebox Handler and can be used to pass information
// between requests.
type Context struct {
	// Context is a idiomatic way to pass information between requests.
	// More information about context.Context can be found here:
	// https://godoc.org/golang.org/x/net/context
	Context  context.Context
	response http.ResponseWriter
	request  *http.Request
	vars     httprouter.Params
	weavebox *Weavebox
}

// Response returns a default http.ResponseWriter
func (c *Context) Response() http.ResponseWriter {
	return c.response
}

// Request returns a default http.Request ptr
func (c *Context) Request() *http.Request {
	return c.request
}

// JSON is a helper function for writing a JSON encoded representation of v to
// the ResponseWriter.
func (c *Context) JSON(code int, v interface{}) error {
	c.Response().Header().Set("Content-Type", "application/json")
	c.Response().WriteHeader(code)
	return json.NewEncoder(c.Response()).Encode(v)
}

// Text is a helper function for writing a text/plain string to the ResponseWriter
func (c *Context) Text(code int, text string) error {
	c.Response().Header().Set("Content-Type", "text/plain")
	c.Response().WriteHeader(code)
	c.Response().Write([]byte(text))
	return nil
}

// DecodeJSON is a helper that decodes the request Body to v.
// For a more in depth use of decoding and encoding JSON, use the std JSON package.
func (c *Context) DecodeJSON(v interface{}) error {
	return json.NewDecoder(c.Request().Body).Decode(v)
}

// Render calls the templateEngines Render function
func (c *Context) Render(name string, data interface{}) error {
	return c.weavebox.templateEngine.Render(c.Response(), name, data)
}

// Param returns the url named parameter given in the route prefix by its name
// 	app.Get("/:name", ..) => ctx.Param("name")
func (c *Context) Param(name string) string {
	return c.vars.ByName(name)
}

// Query returns the url query parameter by its name.
// 	app.Get("/api?limit=25", ..) => ctx.Query("limit")
func (c *Context) Query(name string) string {
	return c.request.URL.Query().Get(name)
}

// Form returns the form parameter by its name
func (c *Context) Form(name string) string {
	return c.request.FormValue(name)
}

// Header returns the request header by name
func (c *Context) Header(name string) string {
	return c.request.Header.Get(name)
}

// Redirect redirects the request to the provided URL with the given status code.
func (c *Context) Redirect(url string, code int) error {
	if code < http.StatusMultipleChoices || code > http.StatusTemporaryRedirect {
		return errors.New("invalid redirect code")
	}
	http.Redirect(c.response, c.request, url, code)
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

// Renderer renders any kind of template. Weavebox allows the use of different
// template engines, if they implement the Render method.
type Renderer interface {
	Render(w io.Writer, name string, data interface{}) error
}
