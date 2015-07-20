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

var defaultErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

type Weavebox struct {
	ErrorHandler    ErrorHandlerFunc
	NotFoundHandler http.Handler
	Output          io.Writer
	EnableLog       bool
	router          *httprouter.Router
	middleware      []Handler
	prefix          string
}

func New() *Weavebox {
	return &Weavebox{
		router:    httprouter.New(),
		Output:    os.Stderr,
		EnableLog: true,
	}
}

func (w *Weavebox) Serve(port int) error {
	w.init()
	portStr := fmt.Sprintf(":%d", port)
	fmt.Fprintf(w.Output, "app listening on 0.0.0.0:%d\n", port)
	return http.ListenAndServe(portStr, w)
}

func (w *Weavebox) Get(route string, h Handler) {
	w.add("GET", route, h)
}

func (w *Weavebox) Post(route string, h Handler) {
	w.add("POST", route, h)
}

func (w *Weavebox) Put(route string, h Handler) {
	w.add("PUT", route, h)
}

func (w *Weavebox) Delete(route string, h Handler) {
	w.add("DELETE", route, h)
}

func (w *Weavebox) Static(prefix, dir string) {
	w.router.ServeFiles(path.Join(prefix, "*filepath"), http.Dir(dir))
}

func (w *Weavebox) Use(handlers ...Handler) {
	for _, h := range handlers {
		w.middleware = append(w.middleware, h)
	}
}

func (w *Weavebox) Subrouter(prefix string) *Box {
	b := &Box{*w}
	b.Weavebox.prefix += prefix
	return b
}

func (w *Weavebox) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	if w.EnableLog {
		start := time.Now()
		logger := &responseLogger{w: rw}
		w.router.ServeHTTP(logger, r)
		w.writeLog(r, start, logger.Status(), logger.Size())
		// saves a allocation by seperating the whole logger if log is disabled
	} else {
		w.router.ServeHTTP(rw, r)
	}
}

type Box struct {
	Weavebox
}

func (b *Box) Reset() *Box {
	b.Weavebox.middleware = nil
	return b
}

func (w *Weavebox) init() {
	if w.ErrorHandler == nil {
		w.ErrorHandler = defaultErrorHandler
	}
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

type Handler func(ctx *Context, w http.ResponseWriter, r *http.Request) error

type ErrorHandlerFunc func(w http.ResponseWriter, r *http.Request, err error)

type Context struct {
	Context context.Context
	Vars    httprouter.Params
}

func JSON(w http.ResponseWriter, code int, v interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	return json.NewEncoder(w).Encode(v)
}

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
