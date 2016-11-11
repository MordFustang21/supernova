# Nova
An express like framework for go web servers

Provides a lot of the same methods and functionality as express.js

Example
```go
s := supernova.Super()

//Static folder example
s.AddStatic("/sitedir/")
//If you want to cache a file (seconds)
s.SetCacheTimeout(5)

//Middleware Example
s.Use(func(req *supernova.Request, next func()) {
    res.R.Header().Set("Powered-By", "Nova")
    next()
})

//Route Examples
s.Get("/test/taco/:apple", func(req *supernova.Request) {
    type test struct {
        Apple string
    }

    testS := test{}
    err := req.Json(&testS)
    if err != nil {
        log.Println(err)
    }
    res.Send("Received Taco")
});

s.Get("/test/:taco/:apple", func(req *supernova.Request) {
    res.Json(req.RouteParams)
});

err := s.Serve(":8080")

if err != nil {
    log.Fatal(err)
}
```
# Todo:
Add graceful stopping