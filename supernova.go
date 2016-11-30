package supernova

import (
	"bytes"
	"compress/gzip"
	"github.com/valyala/fasthttp"
	"io/ioutil"
	"log"
	"mime"
	"os"
	"strings"
	"sync"
	"time"
)

type SuperNova struct {
	paths              map[string]Route
	getPaths           map[string]Route
	postPaths          map[string]Route
	putPaths           map[string]Route
	deletePaths        map[string]Route
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
		switch string(request.Method()) {
		case "GET":
			lookupPaths = sn.getPaths
			break
		case "PUT":
			lookupPaths = sn.putPaths
			break
		case "POST":
			lookupPaths = sn.postPaths
			break
		case "DELETE":
			lookupPaths = sn.deletePaths
			break
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
		route, ok = sn.paths[path]
		if ok {
			route.rq = request

			//Prepare data for call
			route.prepare()

			//Call user handler
			route.call()
			return
		}

		_, pathParts = pathParts[len(pathParts)-1], pathParts[:len(pathParts)-1]
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

	if sn.paths == nil {
		sn.paths = make(map[string]Route, 0)
	}

	routeObj := buildRoute(route, routeFunc)
	sn.paths[routeObj.route] = routeObj
}

func (sn *SuperNova) Get(route string, routeFunc func(*Request)) {
	if sn.getPaths == nil {
		sn.getPaths = make(map[string]Route)
	}

	routeObj := buildRoute(route, routeFunc)
	sn.getPaths[routeObj.route] = routeObj
}

func (sn *SuperNova) Post(route string, routeFunc func(*Request)) {
	if sn.postPaths == nil {
		sn.postPaths = make(map[string]Route)
	}

	routeObj := buildRoute(route, routeFunc)
	sn.postPaths[routeObj.route] = routeObj
}

func (sn *SuperNova) Put(route string, routeFunc func(*Request)) {
	if sn.putPaths == nil {
		sn.putPaths = make(map[string]Route)
	}

	routeObj := buildRoute(route, routeFunc)
	sn.putPaths[routeObj.route] = routeObj
}

func (sn *SuperNova) Delete(route string, routeFunc func(*Request)) {
	if sn.deletePaths == nil {
		sn.deletePaths = make(map[string]Route)
	}

	routeObj := buildRoute(route, routeFunc)
	sn.deletePaths[routeObj.route] = routeObj
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
		path := staticDir + string(req.Request.RequestURI())

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

			if !ok || time.Now().Unix()-cachedObj.timeCached.Unix() > sn.maxCachedTime {
				contents, err := ioutil.ReadFile(path)
				if err != nil {
					log.Println("unable to read file", err)
				}
				cachedObj = &CachedObj{data: contents, timeCached: time.Now()}
				sn.cachedStatic.files[path] = cachedObj
			}

			sn.cachedStatic.mutex.Unlock()

			if err != nil {
				log.Println("Unable to read file")
			}

			//Set mime type
			extensionParts := strings.Split(path, ".")
			ext := extensionParts[len(extensionParts)-1]
			mType := mime.TypeByExtension("." + ext)

			if mType != "" {
				req.Response.Header.Set("Content-Type", mType)
			}

			if sn.compressionEnabled {
				var b bytes.Buffer
				w := gzip.NewWriter(&b)
				w.Write(cachedObj.data)
				w.Close()
				cachedObj.data = b.Bytes()
				req.Response.Header.Set("Content-Encoding", "gzip")
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
