package main

import "github.com/twanies/weavebox"

func main() {
	app := weavebox.New()

	t := weavebox.NewTemplateEngine("pages")
	// Set single templates
	t.SetTemplates("index.html")
	// Set templates that have a layout
	t.SetTemplatesWithLayout("layout.html", "user/index.html")
	t.Init()
	app.SetTemplateEngine(t)

	app.Get("/", renderIndex)
	app.Get("/user", renderUserDetail)
	app.Serve(3000)
}

func renderIndex(ctx *weavebox.Context) error {
	return ctx.Render("index.html", nil)
}

func renderUserDetail(ctx *weavebox.Context) error {
	username := "anthony"
	return ctx.Render("user/index.html", username)
}
