# weavebox
Opinion based minimalistic web framework for the Go programming language

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

    app := weavebox.New()
    app.Get("/foo", fooHandler)
    app.Post("/bar", barHandler)
    app.Use(authenticate)
    app.Serve(8080)
More complete examples can be found in the examples folder

## Routes

## Static files

## Handlers

## Context

## View / Templates

## Logging

## Helpers


