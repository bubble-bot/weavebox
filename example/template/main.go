package main

import (
	"net/http"

	"github.com/twanies/weavebox"
)

func main() {
	app := weavebox.New()

	t := weavebox.NewTemplateEngine()
	t.SetTemplatesWithLayout("pages/layout.html", "pages/index.html")
	t.Init()
	app.SetTemplateEngine(t)

	app.Get("/", renderIndex)
	app.Serve(3000)
}

func renderIndex(ctx *weavebox.Context, w http.ResponseWriter, r *http.Request) error {
	return ctx.Render(w, "pages/index.html", nil)
}
