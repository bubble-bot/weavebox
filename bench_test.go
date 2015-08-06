package weavebox

import (
	"net/http"
	"testing"
)

func BenchmarkGetWithValues(b *testing.B) {
	app := New()
	app.EnableLog = false
	app.Get("/hello/:name", func(ctx *Context) error { return nil })

	for i := 0; i < b.N; i++ {
		r, err := http.NewRequest("GET", "/hello/anthony", nil)
		if err != nil {
			panic(err)
		}
		app.ServeHTTP(nil, r)
	}
}

func BenchmarkBoxGetWithValues(b *testing.B) {
	app := New()
	app.EnableLog = false
	admin := app.Box("/admin")
	admin.Get("/:name", func(ctx *Context) error { return nil })

	for i := 0; i < b.N; i++ {
		r, err := http.NewRequest("GET", "/admin/anthony", nil)
		if err != nil {
			panic(err)
		}
		app.ServeHTTP(nil, r)
	}
}
