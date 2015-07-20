package weavebox

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"

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
	router          *httprouter.Router
	middleware      []Handler
	prefix          string
}

func New() *Weavebox {
	return &Weavebox{router: httprouter.New()}
}

func (w *Weavebox) Serve(port int) error {
	w.init()
	portStr := fmt.Sprintf(":%d", port)
	fmt.Fprintf(w.Output, "app listening on 0.0.0.0:%d\n", port)
	return http.ListenAndServe(portStr, w.router)
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
	if w.Output == nil {
		w.Output = os.Stdout
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
