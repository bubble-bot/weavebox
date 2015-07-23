package weavebox

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
	t.Error("TODO")
}

func TestMiddleware(t *testing.T) {
	t.Error("TODO")
}

func TestBoxMiddlewareReset(t *testing.T) {
	t.Error("TODO")
}

func TestErrorHandler(t *testing.T) {
	t.Error("TODO")
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
