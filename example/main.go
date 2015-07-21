package main

import (
	"net/http"

	"github.com/twanies/weavebox"
)

func main() {
	app := weavebox.New()

	app.Get("/hello/:name", greetingHandler)

	app.Serve(3000)
}

func greetingHandler(ctx *weavebox.Context, w http.ResponseWriter, r *http.Request) error {
	name := ctx.Vars.ByName("name")
	return weavebox.Text(w, http.StatusOK, name)
}
