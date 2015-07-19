package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"golang.org/x/net/context"

	"github.com/twanies/weavebox"
)

func main() {
	app := weavebox.New()

	// Set a custom errorHandler that will handle all errors returned
	// from our handlers.
	app.SetErrorHandler(errorHandler)

	// Set a custom notfound handler
	app.NotFoundHandler = http.HandlerFunc(notFoundHandler)

	// You can setup static routes. This wil serve all files in the root folder
	// when /public/ route is matched.
	app.Static("/public/", "./")

	// simple Sinatra style routing syntax
	app.Get("/hello/:name", greetingHandler)

	// define a admin subrouter
	admin := app.Subrouter("/admin")
	// register some middleware only for admin routes
	admin.Register(authenticate)
	admin.Get("/piggybank", piggyBankHandler)

	log.Fatal(app.Serve(3000))
}

// this is how a default weavebox Handler looks like
func greetingHandler(ctx *weavebox.Context, w http.ResponseWriter, r *http.Request) error {
	name := ctx.Vars.Get(":name")
	greeting := fmt.Sprintf("Hello, %s\nHow are you today?", name)
	return weavebox.Text(w, http.StatusOK, greeting)
}

func piggyBankHandler(ctx *weavebox.Context, w http.ResponseWriter, r *http.Request) error {
	// use the context to get the token passed in the authenticate handler
	user := ctx.Context.Value("from").(string)
	piggyBank := Bank{
		Amount: 100,
		From:   user,
	}

	return weavebox.JSON(w, http.StatusOK, piggyBank)
}

// middleware that will check some credentials and stop untrusted requests
// for the sake of this example we asume a request that has a Authorization
// header set is trusted.
func authenticate(ctx *weavebox.Context, w http.ResponseWriter, r *http.Request) error {
	token := r.Header.Get("Authorization")
	if token == "" {
		// error will be handled by our errorHandler.
		return errors.New("Not Authorized")
	}

	// simulate a user fetched by the given token, for example in some persistent storage.
	user := "trustedJohn"
	// use the powerfull context to pass some information to other middleware.
	ctx.Context = context.WithValue(ctx.Context, "from", user)
	return nil
}

type Bank struct {
	Amount int
	From   string
}

func errorHandler(w http.ResponseWriter, r *http.Request, err error) {
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "YOW, Didnt Found what you looking for!", http.StatusNotFound)
}
