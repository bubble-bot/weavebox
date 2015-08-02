# weavebox [![GoDoc](https://godoc.org/github.com/twanies/weavebox?status.svg)](https://godoc.org/github.com/twanies/weavebox) [![Travis CI](https://travis-ci.org/twanies/weavebox.svg?branch=master)](https://travis-ci.org/twanies/weavebox)
Minimalistic web framework for the Go programming language.

## Installation
`go get github.com/twanies/weavebox`

## Features
- fast route dispatching backed by httprouter
- easy to add middleware handlers
- subrouting with seperated middleware handlers
- central based error handling
- build in template engine
- fast, lightweight and extendable

## Basic usage
    package main
    import "github.com/twanies/weavebox"

    func main() {
        app := weavebox.New()

        app.Get("/foo", fooHandler)
        app.Post("/bar", barHandler)
        app.Use(middleware1, middleware2)

        friends := app.Subrouter("/friends")
        friends.Get("/profile", profileHandler)
        friends.Use(middleware3, middleware4)
        
        app.Serve(8080)
    }
More complete examples can be found in the examples folder

## Routes
    app := weavebox.New()

    app.Get("/", func(ctx *weavebox.Context) error {
       .. do something .. 
    })
    app.Post("/", func(ctx *weavebox.Context) error {
       .. do something .. 
    })
    app.Put("/", func(ctx *weavebox.Context) error {
       .. do something .. 
    })
    app.Delete("/", func(ctx *weavebox.Context) error {
       .. do something .. 
    })

get named url parameters

    app.Get("/hello/:name", func(ctx *weavebox.Context) error {
        name := ctx.Param("name")
    })

## Static files
Make our assets are accessable trough /assets/styles.css

    app := weavebox.New()
    app.Static("/assets", "public/assets")

## Handlers
### A definition of a weavebox.Handler

    func(ctx *weavebox.Context) error

Weavebox only accepts handlers of type `weavebox.Handler` to be passed as functions in routes. You can convert any type of handler to a `weavebox.Handler`.

    func myHandler(name string) weavebox.Handler{
        .. do something ..
       return func(ctx *weavebox.Context) error {
            return ctx.Text(w, http.StatusOK, name)
       }
    }

### Returning errors
Each handler requires an error to be returned. This is personal idiom but it brings some benifits for handling your errors inside request handlers.
    
    func someHandler(ctx *weavebox.Context) error {
        // simple error handling by returning all errors 
        err := someFunc(); err != nil {
            return err
        }
        ...
        req, err := http.NewRequest(...)
        if err != nil {
            return err
        }
    }

A weavebox ErrorHandlerFunc    

    func(ctx *weavebox.Context, err error)
    
Handle all errors returned by adding a custom errorHandler for our application.

    app := weavebox.New()
    app.ErrorHandler = func(ctx *weavebox.Context, err error) {
        .. handle the error ..
    }

## Context
Context is a request based object helping you with a series of functions performed against the current request scope.

### Passing values arround middleware functions
Context provides a context.Context for passing request scoped values arround middleware functions.

Create a new context and pass some values

    func someMiddleware(ctx *weavebox.Context) error {
        ctx.Context = context.WithValue(ctx.Context, "foo", "bar")
        return someMiddleware2(ctx)
    }

Get the value back from the context in another middleware function

    func someMiddleware2(ctx *weavebox.Context) error {
        value := ctx.Context.Value("foo").(string)
        ..
    }

### Helper functions
Context also provides a series of helper functions like responding JSON en text, JSON decoding etc..
    
    func createUser(ctx *weavebox.Context) error {
        user := model.User{}
        if err := ctx.DecodeJSON(&user); err != nil {
            return errors.New("failed to decode the response body")
        }
        ..
        return ctx.JSON(http.StatusCreated, user)
    }

    func login(ctx *weavebox.Context) error {
        token := ctx.Header("x-hmac-token")
        if token == "" {
            ctx.Redirect("/login", http.StatusMovedPermanently)
            return nil
        }
        ..
    }


## View / Templates

## Logging
