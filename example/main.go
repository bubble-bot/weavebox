package main

import (
	"errors"
	"flag"
	"fmt"
	"net/http"

	"golang.org/x/net/context"

	"github.com/twanies/weavebox"
)

// Simpel example how to use weavebox with a "datastore" by making use
// of weavebox.Context to pass information between middleware and handlers

func main() {
	listen := flag.Int("listen", 3000, "listen address of the application")
	flag.Parse()

	app := weavebox.New()

	// centralizing our errors returned from middleware and request handlers
	app.ErrorHandler = errorHandler

	app.Get("/hello/:name", greetingHandler)
	app.Use(dbContextHandler)

	// make a subrouter and register some middleware for it
	admin := app.Subrouter("/admin")
	admin.Get("/:name", adminGreetingHandler)
	admin.Use(authenticate)

	app.Serve(*listen)
}

type datastore struct {
	name string
}

type dbContext struct {
	context.Context
	ds *datastore
}

func (c *dbContext) Value(key interface{}) interface{} {
	if key == "datastore" {
		return c.ds
	}
	return c.Context.Value(key)
}

func newDatastoreContext(parent context.Context, ds *datastore) context.Context {
	return &dbContext{parent, ds}
}
func dbContextHandler(ctx *weavebox.Context, w http.ResponseWriter, r *http.Request) error {
	db := datastore{"mydatabase"}
	ctx.Context = newDatastoreContext(ctx.Context, &db)
	return nil
}

// Only the powerfull have access to the admin routes
func authenticate(ctx *weavebox.Context, w http.ResponseWriter, r *http.Request) error {
	admins := []string{"toby", "master iy", "c.froome"}
	name := ctx.Vars.ByName("name")

	for _, admin := range admins {
		if admin == name {
			return nil
		}
	}
	return errors.New("access forbidden")
}

// context helper function to stay lean and mean in your handlers
func datastoreFromContext(ctx context.Context) *datastore {
	return ctx.Value("datastore").(*datastore)
}

func greetingHandler(ctx *weavebox.Context, w http.ResponseWriter, r *http.Request) error {
	name := ctx.Vars.ByName("name")
	db := datastoreFromContext(ctx.Context)
	greeting := fmt.Sprintf("Greetings, %s\nYour database %s is ready", name, db.name)
	return weavebox.Text(w, http.StatusOK, greeting)
}

func adminGreetingHandler(ctx *weavebox.Context, w http.ResponseWriter, r *http.Request) error {
	name := ctx.Vars.ByName("name")
	db := datastoreFromContext(ctx.Context)
	greeting := fmt.Sprintf("Greetings powerfull admin, %s\nYour database %s is ready", name, db.name)
	return weavebox.Text(w, http.StatusOK, greeting)
}

// custom centralized error handling
func errorHandler(w http.ResponseWriter, r *http.Request, err error) {
	http.Error(w, "Hey some error occured: "+err.Error(), http.StatusInternalServerError)
}
