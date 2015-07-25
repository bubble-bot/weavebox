package weavebox

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/net/context"
)

var noopHandler = func(ctx *Context, w http.ResponseWriter, r *http.Request) error { return nil }

func TestMethodGet(t *testing.T) {
	w := New()
	w.Get("/", noopHandler)
	code, _ := doRequest(t, "GET", "/", nil, w)
	isHTTPStatusOK(t, code)
}

func TestMethodPost(t *testing.T) {
	w := New()
	w.Post("/", noopHandler)
	code, _ := doRequest(t, "POST", "/", nil, w)
	isHTTPStatusOK(t, code)
}

func TestMethodPut(t *testing.T) {
	w := New()
	w.Put("/", noopHandler)
	code, _ := doRequest(t, "PUT", "/", nil, w)
	isHTTPStatusOK(t, code)
}

func TestMethodDelete(t *testing.T) {
	w := New()
	w.Delete("/", noopHandler)
	code, _ := doRequest(t, "DELETE", "/", nil, w)
	isHTTPStatusOK(t, code)
}

func TestSubrouter(t *testing.T) {
	w := New()
	sr := w.Subrouter("/foo")
	sr.Get("/bar", noopHandler)
	code, _ := doRequest(t, "GET", "/foo/bar", nil, w)
	isHTTPStatusOK(t, code)
}

func TestStatic(t *testing.T) {
	w := New()
	w.Static("/public", "./")
	code, body := doRequest(t, "GET", "/public/README.md", nil, w)
	isHTTPStatusOK(t, code)
	if len(body) == 0 {
		t.Error("body cannot be empty")
	}
	if !strings.Contains(body, "weavebox") {
		t.Error("expecting body containing string (weavebox)")
	}

	code, body = doRequest(t, "GET", "/public/nofile", nil, w)
	if code != http.StatusNotFound {
		t.Error("expecting status 404 got %d", code)
	}
}

func TestContext(t *testing.T) {
	w := New()
	w.Get("/", checkContext(t, "m1", "m1"))
	w.Use(func(ctx *Context, w http.ResponseWriter, r *http.Request) error {
		ctx.Context = context.WithValue(ctx.Context, "m1", "m1")
		return nil
	})
	code, _ := doRequest(t, "GET", "/", nil, w)
	isHTTPStatusOK(t, code)

	w.Get("/some", checkContext(t, "m1", "m2"))
	w.Use(func(ctx *Context, w http.ResponseWriter, r *http.Request) error {
		ctx.Context = context.WithValue(ctx.Context, "m1", "m2")
		w.WriteHeader(http.StatusBadRequest)
		return nil
	})
	code, _ = doRequest(t, "GET", "/some", nil, w)
	if code != http.StatusBadRequest {
		t.Error("expecting %d, got %d", http.StatusBadRequest, code)
	}
}

func checkContext(t *testing.T, key, expect string) Handler {
	return func(ctx *Context, w http.ResponseWriter, r *http.Request) error {
		value := ctx.Context.Value(key).(string)
		if value != expect {
			t.Errorf("expected %s got %s", expect, value)
		}
		return nil
	}
}

func TestMiddleware(t *testing.T) {
	buf := &bytes.Buffer{}
	w := New()
	w.Use(func(ctx *Context, w http.ResponseWriter, r *http.Request) error {
		buf.WriteString("a")
		return nil
	})
	w.Use(func(ctx *Context, w http.ResponseWriter, r *http.Request) error {
		buf.WriteString("b")
		return nil
	})
	w.Use(func(ctx *Context, w http.ResponseWriter, r *http.Request) error {
		buf.WriteString("c")
		return nil
	})
	w.Use(func(ctx *Context, w http.ResponseWriter, r *http.Request) error {
		buf.WriteString("d")
		return nil
	})
	w.Get("/", noopHandler)
	code, _ := doRequest(t, "GET", "/", nil, w)
	isHTTPStatusOK(t, code)
	if buf.String() != "abcd" {
		t.Error("expecting abcd got %s", buf.String())
	}
}

func TestBoxMiddlewareReset(t *testing.T) {
	buf := &bytes.Buffer{}
	w := New()
	w.Use(func(ctx *Context, w http.ResponseWriter, r *http.Request) error {
		buf.WriteString("a")
		return nil
	})
	w.Use(func(ctx *Context, w http.ResponseWriter, r *http.Request) error {
		buf.WriteString("b")
		return nil
	})
	sub := w.Subrouter("/sub").Reset()
	sub.Get("/", noopHandler)
	code, _ := doRequest(t, "GET", "/sub", nil, w)
	isHTTPStatusOK(t, code)
	if buf.String() != "" {
		t.Error("expecting empty buffer got %s", buf.String())
	}
}

func TestBoxMiddlewareInheritsParent(t *testing.T) {
	buf := &bytes.Buffer{}
	w := New()
	w.Use(func(ctx *Context, w http.ResponseWriter, r *http.Request) error {
		buf.WriteString("a")
		return nil
	})
	w.Use(func(ctx *Context, w http.ResponseWriter, r *http.Request) error {
		buf.WriteString("b")
		return nil
	})
	sub := w.Subrouter("/sub")
	sub.Get("/", noopHandler)
	code, _ := doRequest(t, "GET", "/sub", nil, w)
	isHTTPStatusOK(t, code)
	if buf.String() != "ab" {
		t.Error("expecting ab got %s", buf.String())
	}
}

func TestErrorHandler(t *testing.T) {
	w := New()
	errorMsg := "oops! something went wrong"
	w.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		w.WriteHeader(http.StatusInternalServerError)
		if err.Error() != errorMsg {
			t.Error("expecting %s, got %s", errorMsg, err.Error())
		}
	}
	w.Use(func(ctx *Context, w http.ResponseWriter, r *http.Request) error {
		return errors.New(errorMsg)
	})
	w.Get("/", noopHandler)
	code, _ := doRequest(t, "GET", "/", nil, w)
	if code != http.StatusInternalServerError {
		t.Error("expecting code 500 got %s", code)
	}
}

func TestWeaveboxHandler(t *testing.T) {
	w := New()
	handle := func(respStr string) Handler {
		return func(ctx *Context, w http.ResponseWriter, r *http.Request) error {
			return Text(w, http.StatusOK, respStr)
		}
	}
	w.Get("/a", handle("a"))
	w.Get("/b", handle("b"))
	w.Get("/c", handle("c"))

	for _, r := range []string{"a", "b", "c"} {
		code, body := doRequest(t, "GET", "/"+r, nil, w)
		isHTTPStatusOK(t, code)
		if body != r {
			t.Errorf("expecting %s got %s", r, body)
		}
	}
}

func TestNotFoundHandler(t *testing.T) {
	w := New()
	code, body := doRequest(t, "GET", "/", nil, w)
	if code != http.StatusNotFound {
		t.Errorf("expecting code 404 got %d", code)
	}
	if !strings.Contains(body, "404 page not found") {
		t.Errorf("expecting body: 404 page not found got %s", body)
	}
}

func TestNotFoundHandlerOverride(t *testing.T) {
	w := New()
	notFoundMsg := "hey! not found"
	w.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Text(w, http.StatusNotFound, notFoundMsg)
	})

	// init is called before serve or serveTLS to initialize some data that
	// needs to be passed to the underlaying router. For this test to pass
	// we need to call init to override the default NotFoundHandler
	w.init()
	code, body := doRequest(t, "GET", "/", nil, w)
	if code != http.StatusNotFound {
		t.Errorf("expecting code 404 got %d", code)
	}
	if !strings.Contains(body, notFoundMsg) {
		t.Errorf("expecting body: %s got %s", notFoundMsg, body)
	}
}

func isHTTPStatusOK(t *testing.T, code int) {
	if code != http.StatusOK {
		t.Errorf("Expecting status 200 got %d", code)
	}
}

func doRequest(t *testing.T, method, route string, body io.Reader, w *Weavebox) (int, string) {
	r, err := http.NewRequest(method, route, body)
	if err != nil {
		t.Fatal(err)
	}
	rw := httptest.NewRecorder()
	w.ServeHTTP(rw, r)
	return rw.Code, rw.Body.String()
}
