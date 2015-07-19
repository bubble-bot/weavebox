// original can be found github.com/bmizerany/pat
package weavebox

import (
	"net/http"
	"net/url"
	"strings"
)

// Router is an HTTP request multiplexer. It matches the URL of each
// incoming request against a list of registered patterns with their associated
// methods and calls the handler for the pattern that most closely matches the
// URL.
//
// Pattern matching attempts each pattern in the order in which they were
// registered.
//
// Patterns may contain literals or captures. Capture names start with a colon
// and consist of letters A-Z, a-z, _, and 0-9. The rest of the pattern
// matches literally. The portion of the URL matching each name ends with an
// occurrence of the character in the pattern immediately following the name,
// or a /, whichever comes first. It is possible for a name to match the empty
// string.
//
// Example pattern with one capture:
//   /hello/:name
// Will match:
//   /hello/blake
//   /hello/keith
// Will not match:
//   /hello/blake/
//   /hello/blake/foo
//   /foo
//   /foo/bar
//
// Example 2:
//    /hello/:name/
// Will match:
//   /hello/blake/
//   /hello/keith/foo
//   /hello/blake
//   /hello/keith
// Will not match:
//   /foo
//   /foo/bar
//
// A pattern ending with a slash will get an implicit redirect to it's
// non-slash version.  For example: Get("/foo/", handler) will implicitly
// register Get("/foo", handler). You may override it by registering
// Get("/foo", anotherhandler) before the slash version.
//
// Retrieve the capture from the r.URL.Query().Get(":name") in a handler (note
// the colon). If a capture name appears more than once, the additional values
// are appended to the previous values (see
// http://golang.org/pkg/net/url/#Values)
//
// When "Method Not Allowed":
//
// Pat knows what methods are allowed given a pattern and a URI. For
// convenience, PatternServeMux will add the Allow header for requests that
// match a pattern for a method other than the method requested and set the
// Status to "405 Method Not Allowed".
type Router struct {
	// NotFoundHandler is a fallback handler when no route is matched
	NotFoundHandler http.Handler

	handlers map[string][]*patHandler
}

// NewRouter returns a new PatternServeMux.
func NewRouter() *Router {
	return &Router{handlers: make(map[string][]*patHandler)}
}

// ServeHTTP matches r.URL.Path against its routing table using the rules
// described above.
func (router *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, ph := range router.handlers[r.Method] {
		if params, ok := ph.try(r.URL.Path); ok {
			if len(params) > 0 {
				r.URL.RawQuery = url.Values(params).Encode() + "&" + r.URL.RawQuery
			}
			ph.ServeHTTP(w, r)
			return
		}
	}

	allowed := make([]string, 0, len(router.handlers))
	for meth, handlers := range router.handlers {
		if meth == r.Method {
			continue
		}

		for _, ph := range handlers {
			if _, ok := ph.try(r.URL.Path); ok {
				allowed = append(allowed, meth)
			}
		}
	}

	if len(allowed) == 0 {
		if router.NotFoundHandler != nil {
			router.NotFoundHandler.ServeHTTP(w, r)
			return
		}
		http.NotFound(w, r)
		return
	}

	w.Header().Add("Allow", strings.Join(allowed, ", "))
	http.Error(w, "Method Not Allowed", 405)
}

// Head will register a pattern with a handler for HEAD requests.
func (router *Router) Head(pat string, h http.Handler) {
	router.Add("HEAD", pat, h)
}

// Get will register a pattern with a handler for GET requests.
// It also registers pat for HEAD requests. If this needs to be overridden, use
// Head before Get with pat.
func (router *Router) Get(pat string, h http.Handler) {
	router.Add("HEAD", pat, h)
	router.Add("GET", pat, h)
}

// Post will register a pattern with a handler for POST requests.
func (router *Router) Post(pat string, h http.Handler) {
	router.Add("POST", pat, h)
}

// Put will register a pattern with a handler for PUT requests.
func (router *Router) Put(pat string, h http.Handler) {
	router.Add("PUT", pat, h)
}

// Del will register a pattern with a handler for DELETE requests.
func (router *Router) Del(pat string, h http.Handler) {
	router.Add("DELETE", pat, h)
}

// Options will register a pattern with a handler for OPTIONS requests.
func (router *Router) Options(pat string, h http.Handler) {
	router.Add("OPTIONS", pat, h)
}

// Add will register a pattern with a handler for meth requests.
func (router *Router) Add(meth, pat string, h http.Handler) {
	router.handlers[meth] = append(router.handlers[meth], &patHandler{pat, h})

	n := len(pat)
	if n > 0 && pat[n-1] == '/' {
		router.Add(meth, pat[:n-1], http.RedirectHandler(pat, http.StatusMovedPermanently))
	}
}

// Tail returns the trailing string in path after the final slash for a pat ending with a slash.
//
// Examples:
//
//	Tail("/hello/:title/", "/hello/mr/mizerany") == "mizerany"
//	Tail("/:a/", "/x/y/z")                       == "y/z"
//
func Tail(pat, path string) string {
	var i, j int
	for i < len(path) {
		switch {
		case j >= len(pat):
			if pat[len(pat)-1] == '/' {
				return path[i:]
			}
			return ""
		case pat[j] == ':':
			var nextc byte
			_, nextc, j = match(pat, isAlnum, j+1)
			_, _, i = match(path, matchPart(nextc), i)
		case path[i] == pat[j]:
			i++
			j++
		default:
			return ""
		}
	}
	return ""
}

type patHandler struct {
	pat string
	http.Handler
}

func (ph *patHandler) try(path string) (url.Values, bool) {
	p := make(url.Values)
	var i, j int
	for i < len(path) {
		switch {
		case j >= len(ph.pat):
			if ph.pat != "/" && len(ph.pat) > 0 && ph.pat[len(ph.pat)-1] == '/' {
				return p, true
			}
			return nil, false
		case ph.pat[j] == ':':
			var name, val string
			var nextc byte
			name, nextc, j = match(ph.pat, isAlnum, j+1)
			val, _, i = match(path, matchPart(nextc), i)
			p.Add(":"+name, val)
		case path[i] == ph.pat[j]:
			i++
			j++
		default:
			return nil, false
		}
	}
	if j != len(ph.pat) {
		return nil, false
	}
	return p, true
}

func matchPart(b byte) func(byte) bool {
	return func(c byte) bool {
		return c != b && c != '/'
	}
}

func match(s string, f func(byte) bool, i int) (matched string, next byte, j int) {
	j = i
	for j < len(s) && f(s[j]) {
		j++
	}
	if j < len(s) {
		next = s[j]
	}
	return s[i:j], next, j
}

func isAlpha(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_'
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

func isAlnum(ch byte) bool {
	return isAlpha(ch) || isDigit(ch)
}
