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
	Paths              []Route
	GetPaths           []Route
	PostPaths          []Route
	PutPaths           []Route
	DeletePaths        []Route
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

	var lookupPaths []Route

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

		for routeIndex := range lookupPaths {
			route := lookupPaths[routeIndex]
			if route.route == path || route.route == path + "/" {
				route.rq = request

				//Prepare data for call
				route.prepare()

				//Call user handler
				route.call()
				return
			}
		}

		//TODO: Remove duplicate code
		for routeIndex := range sn.Paths {
			route := sn.Paths[routeIndex]
			if route.route == path || route.route == path + "/" {
				route.rq = request

				//Prepare data for call
				route.prepare()

				//Call user handler
				route.call()
				return
			}
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
		sn.Paths = make([]Route, 0)
	}

	sn.Paths = append(sn.Paths, *buildRoute(route, routeFunc))
}

func (sn *SuperNova) Get(route string, routeFunc func(*Request)) {
	if sn.GetPaths == nil {
		sn.GetPaths = make([]Route, 0)
	}

	sn.GetPaths = append(sn.GetPaths, *buildRoute(route, routeFunc))
}

func (sn *SuperNova) Post(route string, routeFunc func(*Request)) {
	if sn.PostPaths == nil {
		sn.PostPaths = make([]Route, 0)
	}

	sn.PostPaths = append(sn.PostPaths, *buildRoute(route, routeFunc))
}

func (sn *SuperNova) Put(route string, routeFunc func(*Request)) {
	if sn.PutPaths == nil {
		sn.PutPaths = make([]Route, 0)
	}

	sn.PutPaths = append(sn.PutPaths, *buildRoute(route, routeFunc))
}

func (sn *SuperNova) Delete(route string, routeFunc func(*Request)) {
	if sn.DeletePaths == nil {
		sn.DeletePaths = make([]Route, 0)
	}

	sn.DeletePaths = append(sn.DeletePaths, *buildRoute(route, routeFunc))
}

func buildRoute(route string, routeFunc func(*Request)) *Route {
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

	routeObj.route = baseDir

	return routeObj
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