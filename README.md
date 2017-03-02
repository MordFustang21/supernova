# Supernova
[![GoDoc](https://godoc.org/github.com/MordFustang21/supernova?status.svg)](https://godoc.org/github.com/MordFustang21/supernova)
[![Go Report Card](https://goreportcard.com/badge/github.com/mordfustang21/supernova)](https://goreportcard.com/report/github.com/mordfustang21/supernova)
[![Build Status](https://travis-ci.org/MordFustang21/supernova.svg?branch=master)](https://travis-ci.org/MordFustang21/supernova)

An express like router for fasthttp

Provides a lot of the same methods and functionality as Expressjs

Example
```go
package main

import (
	"log"
	"github.com/MordFustang21/supernova"
)

func main() {
	s := supernova.Super()

	//Static folder example
	s.AddStatic("/sitedir/")
	//If you want to cache a file (seconds)
	s.SetCacheTimeout(5)

	//Middleware Example
	s.Use(func(req *supernova.Request, next func()) {
		req.Response.Header.Set("Powered-By", "supernova")
		next()
	})

	//Route Examples
	s.Get("/test/taco/:apple", func(req *supernova.Request) {
		type test struct {
			Apple string
		}

		testS := test{}
		err := req.JSON(&testS)
		if err != nil {
			log.Println(err)
		}
		req.Send("Received data")
	});

	s.Get("/test/:taco/:apple", func(req *supernova.Request) {
		req.SendJSON(req.RouteParams)
	});

	err := s.Serve(":8080")

	if err != nil {
		log.Fatal(err)
	}
}
```