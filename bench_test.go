package weavebox

import (
	"net/http"
	"testing"
)

func BenchmarkGetWithValues(b *testing.B) {
	app := New()
	app.Get("/hello/:name", func(ctx *Context, w http.ResponseWriter, r *http.Request) error { return nil })

	for i := 0; i < b.N; i++ {
		r, err := http.NewRequest("GET", "/hello/anthony", nil)
		if err != nil {
			panic(err)
		}
		app.router.ServeHTTP(nil, r)
	}
}

func BenchmarkSubrouterGetWithValues(b *testing.B) {
	app := New()
	admin := app.Subrouter("/admin")
	admin.Get("/:name", func(ctx *Context, w http.ResponseWriter, r *http.Request) error { return nil })

	for i := 0; i < b.N; i++ {
		r, err := http.NewRequest("GET", "/admin/anthony", nil)
		if err != nil {
			panic(err)
		}
		app.router.ServeHTTP(nil, r)
	}

}
