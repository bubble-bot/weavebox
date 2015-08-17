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

        friends := app.Box("/friends")
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

## Box (subrouting)
Box lets you manage routes, contexts and middleware separate from each other.

Create a new weavebox object and attach some middleware and context to it.

    app := weavebox.New()
    app.BindContext(context.WithValue(context.Background(), "foo", "bar")
    app.Get("/", somHandler)
    app.Use(middleware1, middleware2)

Create a box and attach its own middleware and context to it
    
    friends := app.Box("/friends")
    app.BindContext(context.WithValue(context.Background(), "friend1", "john")
    friends.Post("/create", someHandler)
    friends.Use(middleware3, middleware4)

In this case box friends will inherit middleware1 and middleware2 from its parent app. We can reset the middleware from its parent by calling `Reset()`
    
    friends := app.Box("/friends").Reset()
    friends.Use(middleware3, middleware4)

Now box friends will have only middleware3 and middleware4 attached.

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
    errHandler := func(ctx *weavebox.Context, err error) {
        .. handle the error ..
    }
    app.SetErrorHandler(errHandler)

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

### Binding a context
In some cases you want to intitialize a context from the the main function, like a datastore for example. You can set a context out of a request scope by calling `BindContext()`.
    
    app.BindContext(context.WithValue(context.Background(), "foo", "bar"))

As mentioned in the Box section, you can add different contexts to different boxes.
    
    mybox := app.Box("/foo", ..)
    mybox.BindContext(..)

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
### Access Log
Weavebox provides an access-log in an Apache log format for each incomming request. The access-log is disabled by default, to enable the access-log set `app.EnableAccessLog = true`.

`127.0.0.1 - frank [10/Oct/2000:13:55:36 -0700] "GET /apache_pb.gif HTTP/1.0" 200 2326`

### Logging errors and information

## Server
Weavebox HTTP server is a wrapper arround the default std HTTP server, the only difference is that it provides a gracefull shutdown. Weavebox provides both HTTP and HTTPS (TLS).
    
    app := weavebox.New()
    app.ServeTLS(8080, cert, key)
    // or 
    app.Serve(8080)

### Gracefull stopping a weavebox app
Gracefull stopping a weavebox app is done by sending one of these signals to the process.
- SIGINT
- SIGQUIT
- SIGTERM

You can also force-quit your app by sending it `SIGKILL` signal

SIGUSR2 signal is not yet implemented. Reloading a new binary by forking the main process is something that wil be implemented when the need for it is there. Feel free to give some feedback on this feature if you think it can provide a bonus to the package.
