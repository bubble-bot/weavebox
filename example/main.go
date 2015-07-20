package main

import (
	"log"
	"net/http"

	"github.com/twanies/weavebox"
)

func main() {
	app := weavebox.New()
	app.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	app.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found boy?", http.StatusNotFound)
		return
	})

	app.Static("/public", "./")

	app.Get("/hello/:name", testHandler)
	app.Use(tstMiddleware)

	admin := app.Subrouter("/admin")
	admin.Get("/backend", testHandler)
	admin.Use(adminHandler)

	cart := app.Subrouter("/cart").Reset()
	cart.Get("/amount", cartHandler)
	cart.Use(cartMiddleware)

	app.Serve(3000)
}

func cartHandler(ctx *weavebox.Context, w http.ResponseWriter, r *http.Request) error {
	name := ctx.Vars.ByName("name")
	weavebox.Text(w, http.StatusOK, "in cart"+name)
	return nil
}

func cartMiddleware(ctx *weavebox.Context, w http.ResponseWriter, r *http.Request) error {
	log.Println("cart middleware")
	return nil
}

func testHandler(ctx *weavebox.Context, w http.ResponseWriter, r *http.Request) error {
	name := ctx.Vars.ByName("name")
	weavebox.Text(w, http.StatusOK, name)
	return nil
}

func tstMiddleware(ctx *weavebox.Context, w http.ResponseWriter, r *http.Request) error {
	log.Println("tst middleware")
	return nil
}

func adminHandler(ctx *weavebox.Context, w http.ResponseWriter, r *http.Request) error {
	log.Println("in admin handler")
	return nil
}
