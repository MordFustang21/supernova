package superNova

import (
	"github.com/valyala/fasthttp"
	"strings"
)

type SuperNova struct {
	Paths []Route
}

func (sn *SuperNova) Serve(addr string) {
	fasthttp.ListenAndServe(addr, sn.handler)
}

func (sn *SuperNova) handler(ctx *fasthttp.RequestCtx) {
	request := NewRequest(ctx)

	pathParts := strings.Split(string(ctx.Request.RequestURI()), "/")
	path := strings.Join(pathParts, "/")

	for _ = range pathParts {
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
	println("Not found")
	ctx.Error("not found", fasthttp.StatusNotFound)
}

func (sn *SuperNova) AddRoute(route string, routeFunc func(*Request)) {
	//Build route and assign function
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

	if sn.Paths == nil {
		sn.Paths = make([]Route, 0)
	}

	sn.Paths = append(sn.Paths, *routeObj)
}