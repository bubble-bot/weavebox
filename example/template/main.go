package main

import (
	"net/http"

	"github.com/twanies/weavebox"
)

func main() {
	app := weavebox.New()

	t := weavebox.NewTemplateEngine("pages")
	t.SetTemplatesWithLayout("layout.html", "user/index.html")
	t.Init()
	app.SetTemplateEngine(t)

	app.Get("/", renderIndex)
	app.Get("/user", renderUserDetail)
	app.Serve(3000)
}

func renderIndex(ctx *weavebox.Context, w http.ResponseWriter, r *http.Request) error {
	return ctx.Render(w, "pages/index.html", nil)
}

func renderUserDetail(ctx *weavebox.Context, w http.ResponseWriter, r *http.Request) error {
	username := "anthony"
	return ctx.Render(w, "user/index.html", username)
}
