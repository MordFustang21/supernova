package supernova

import (
	"github.com/valyala/fasthttp"
	"strings"
	"os"
	"time"
	"io/ioutil"
	"log"
	"mime"
	"sync"
	"bytes"
	"compress/gzip"
)

type SuperNova struct {
	Paths              map[string]Route
	GetPaths           map[string]Route
	PostPaths          map[string]Route
	PutPaths           map[string]Route
	DeletePaths        map[string]Route
	staticDirs         []string
	middleWare         []MiddleWare
	cachedStatic       *CachedStatic
	maxCachedTime      int64
	compressionEnabled bool
}

type CachedObj struct {
	data       []byte
	timeCached time.Time
}

type CachedStatic struct {
	mutex sync.Mutex
	files map[string]*CachedObj
}

//Middleware obj to hold functions
type MiddleWare struct {
	middleFunc func(*Request, func())
}

func Super() *SuperNova {
	s := new(SuperNova)
	s.cachedStatic = new(CachedStatic)
	s.cachedStatic.files = make(map[string]*CachedObj)
	return s
}

func (sn *SuperNova) Serve(addr string) error {
	return fasthttp.ListenAndServe(addr, sn.handler)
}

func (sn *SuperNova) handler(ctx *fasthttp.RequestCtx) {
	request := NewRequest(ctx)

	//Run Middleware
	finished := sn.runMiddleware(request)

	if !finished {
		return
	}

	pathParts := strings.Split(string(ctx.Request.RequestURI()), "/")
	path := strings.Join(pathParts, "/")

	var lookupPaths map[string]Route

	for range pathParts {
		switch string(request.Ctx.Method()) {
		case "GET":
			lookupPaths = sn.GetPaths
			break;
		case "PUT":
			lookupPaths = sn.PutPaths
			break;
		case "POST":
			lookupPaths = sn.PostPaths
			break;
		case "DELETE":
			lookupPaths = sn.DeletePaths
			break;
		}

		route, ok := lookupPaths[path]
		if ok {
			route.rq = request

			//Prepare data for call
			route.prepare()

			//Call user handler
			route.call()
			return
		}


		//TODO: Remove duplicate code
		route, ok = sn.Paths[path]
		if ok {
			route.rq = request

			//Prepare data for call
			route.prepare()

			//Call user handler
			route.call()
			return
		}

		_, pathParts = pathParts[len(pathParts) - 1], pathParts[:len(pathParts) - 1]
		path = strings.Join(pathParts, "/")
	}

	//Check for static file
	served := sn.serveStatic(request)

	if served {
		return
	}

	ctx.Error("404 Not Found", fasthttp.StatusNotFound)
}

func (sn *SuperNova) All(route string, routeFunc func(*Request)) {

	if sn.Paths == nil {
		sn.Paths = make(map[string]Route, 0)
	}

	routeObj := buildRoute(route, routeFunc)
	sn.Paths[routeObj.route] = routeObj
}

func (sn *SuperNova) Get(route string, routeFunc func(*Request)) {
	if sn.GetPaths == nil {
		sn.GetPaths = make(map[string]Route)
	}

	routeObj := buildRoute(route, routeFunc)
	println("Adding Route: " + routeObj.route)
	sn.GetPaths[routeObj.route] = routeObj
}

func (sn *SuperNova) Post(route string, routeFunc func(*Request)) {
	if sn.PostPaths == nil {
		sn.PostPaths = make(map[string]Route)
	}

	routeObj := buildRoute(route, routeFunc)
	sn.PostPaths[routeObj.route] = routeObj
}

func (sn *SuperNova) Put(route string, routeFunc func(*Request)) {
	if sn.PutPaths == nil {
		sn.PutPaths = make(map[string]Route)
	}

	routeObj := buildRoute(route, routeFunc)
	sn.PutPaths[routeObj.route] = routeObj
}

func (sn *SuperNova) Delete(route string, routeFunc func(*Request)) {
	if sn.DeletePaths == nil {
		sn.DeletePaths = make(map[string]Route)
	}

	routeObj := buildRoute(route, routeFunc)
	sn.DeletePaths[routeObj.route] = routeObj
}

func buildRoute(route string, routeFunc func(*Request)) Route {
	routeObj := new(Route)
	routeObj.routeFunc = routeFunc

	routeObj.routeParamsIndex = make(map[int]string)

	routeParts := strings.Split(route, "/")
	baseDir := ""
	for i := range routeParts {
		if strings.Contains(routeParts[i], ":") {
			routeParamMod := strings.Replace(routeParts[i], ":", "", 1)
			routeObj.routeParamsIndex[i] = routeParamMod
		} else {
			baseDir += routeParts[i] + "/"
		}
	}

	routeObj.route = strings.TrimSuffix(baseDir, "/")

	return *routeObj
}

func (sn *SuperNova) AddStatic(dir string) {
	if sn.staticDirs == nil {
		sn.staticDirs = make([]string, 0)
	}

	if _, err := os.Stat(dir); err == nil {
		sn.staticDirs = append(sn.staticDirs, dir)
	}
}

func (sn *SuperNova) SetCacheTimeout(seconds int64) {
	sn.maxCachedTime = seconds
}

func (sn *SuperNova) EnableGzip(value bool) {
	sn.compressionEnabled = value
}

func (sn *SuperNova) serveStatic(req *Request) bool {
	for i := range sn.staticDirs {
		staticDir := sn.staticDirs[i]
		path := staticDir + string(req.Ctx.Request.RequestURI())

		//Remove all .. for security TODO: Allow if doesn't go above basedir
		path = strings.Replace(path, "..", "", -1)

		//If ends in / default to index.html
		if strings.HasSuffix(path, "/") {
			path += "index.html"
		}

		if _, err := os.Stat(path); err == nil {
			sn.cachedStatic.mutex.Lock()
			var cachedObj *CachedObj
			cachedObj, ok := sn.cachedStatic.files[path]

			if !ok || time.Now().Unix() - cachedObj.timeCached.Unix() > sn.maxCachedTime {
				contents, err := ioutil.ReadFile(path)
				if err != nil {
					log.Println("unable to read file", err)
				}
				cachedObj = &CachedObj{data:contents, timeCached: time.Now()}
				sn.cachedStatic.files[path] = cachedObj
			}

			sn.cachedStatic.mutex.Unlock()

			if err != nil {
				log.Println("Unable to read file")
			}

			//Set mime type
			extensionParts := strings.Split(path, ".")
			ext := extensionParts[len(extensionParts) - 1]
			mType := mime.TypeByExtension("." + ext)

			if mType != "" {
				req.Ctx.Response.Header.Set("Content-Type", mType)
			}

			if sn.compressionEnabled {
				var b bytes.Buffer
				w := gzip.NewWriter(&b)
				w.Write(cachedObj.data)
				w.Close()
				cachedObj.data = b.Bytes()
				req.Ctx.Response.Header.Set("Content-Encoding", "gzip")
			}

			req.Send(cachedObj.data)
			return true
		}
	}
	return false
}

//Adds a new function to the middleware stack
func (s *SuperNova) Use(f func(*Request, func())) {
	if s.middleWare == nil {
		s.middleWare = make([]MiddleWare, 0)
	}
	middle := new(MiddleWare)
	middle.middleFunc = f
	s.middleWare = append(s.middleWare, *middle)
}

//Internal method that runs the middleware
func (s *SuperNova) runMiddleware(req *Request) bool {
	stackFinished := true
	for m := range s.middleWare {
		nextCalled := false
		s.middleWare[m].middleFunc(req, func() {
			nextCalled = true
		})

		if !nextCalled {
			stackFinished = false
			break
		}
	}

	return stackFinished
}