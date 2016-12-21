package supernova

import (
	"strings"
)

type Route struct {
	rq               *Request
	routeFunc        func(*Request)
	routeParamsIndex map[int]string
	route            string
}

func (r *Route) buildRouteParams() {
	routeParams := r.rq.RouteParams
	pathParts := strings.Split(r.rq.BaseUrl, "/")

	for i := range r.routeParamsIndex {
		name := r.routeParamsIndex[i]
		if i <= len(pathParts)-1 {
			routeParams[name] = pathParts[i]
		}
	}
}

func (r *Route) prepare() {
	r.buildRouteParams()
}

func (r *Route) call() {
	r.routeFunc(r.rq)
}
