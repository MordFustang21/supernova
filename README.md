![Supernova Logo](https://raw.githubusercontent.com/MordFustang21/supernova-logo/master/supernova_banner.png)

[![GoDoc](https://godoc.org/github.com/MordFustang21/supernova?status.svg)](https://godoc.org/github.com/MordFustang21/supernova)
[![Go Report Card](https://goreportcard.com/badge/github.com/mordfustang21/supernova)](https://goreportcard.com/report/github.com/mordfustang21/supernova)
[![Build Status](https://travis-ci.org/MordFustang21/supernova.svg?branch=v2)](https://travis-ci.org/MordFustang21/supernova)

An express like router for fasthttp

Provides a lot of the same methods and functionality as Expressjs

### Start using it
1. Download and install
```
$ go get github.com/MordFustang21/supernova
```
2. Import it into your code
```
import "github.com/MordFustang21/supernova"
```

### Use a vendor tool like dep
1. go get dep
```
$ go get -u github.com/golang/dep/cmd/dep
```
2. cd to project folder and run dep
```
$ dep ensure
```

Refer to [dep](https://github.com/golang/dep) for more information

### Basic Usage
http://localhost:8080/hello
```go
package main

import "github.com/MordFustang21/supernova"

func main() {
	s := supernova.New()
	
	s.Get("/hello", func(request *supernova.Request) (int, error) {
	    return request.Send("world")
	})
	
	s.ListenAndServe(":8080")
}

```
#### Retrieving parameters
http://localhost:8080/hello/world
```go
package main

import "github.com/MordFustang21/supernova"

func main() {
	s := supernova.New()
	
	s.Get("/hello/:text", func(request *supernova.Request) (int, error) {
		t := request.RouteParam("text")
	    return request.Send(t)
	})
	
	s.ListenAndServe(":8080")
}
```

#### Returning Errors
http://localhost:8080/hello
```go
package main

import (
	"net/http"
	"github.com/MordFustang21/supernova"
)

func main() {
	s := supernova.New()
	
	s.Post("/hello", func(request *supernova.Request) (int, error) {
		r := struct {
		 World string
		}{}
		
		// ReadJSON will attempt to unmarshall the json from the request body into the given struct
		err := request.ReadJSON(&r)
		if err != nil {
		    return request.Error(http.StatusBadRequest, "couldn't parse request", err.Error())
		}
		
		// JSON will marshall the given object and marshall into into the response body
		return request.JSON(http.StatusOK, r)
	})
	
	s.ListenAndServe(":8080")
	
}
```