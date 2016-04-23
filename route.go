package superNova

import (
	"strings"
)

type Route struct {
	rq               *Request
	rs               *Response
	rr               RequestResponse
	routeParamsIndex map[int]string
	route            string
}

func (r *Route) buildRouteParams() {
	routeParams := make(map[string]string)
	pathParts := strings.Split(string(r.rq.Ctx.Request.RequestURI()), "/")

	for i := range r.routeParamsIndex {
		name := r.routeParamsIndex[i]
		if i <= len(pathParts) - 1 {
			routeParams[name] = pathParts[i]
		}
	}

	r.rq.RouteParams = routeParams
}

func (r *Route) prepare() {
	r.buildRouteParams()
}

func (r *Route) call() {
	r.rr(r.rq, r.rs)
}
