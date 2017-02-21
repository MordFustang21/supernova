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
	reqParts := strings.Split(r.rq.BaseUrl, "/")
	routeParts := strings.Split(r.route, "/")

	for index, val := range routeParts {
		if len(val) > 0 {
			bVal := []byte(val)
			if bVal[0] == ':' {
				routeParams[string(bVal[1:])] = reqParts[index]
			}
		}
	}
}

func (r *Route) prepare() {
	r.buildRouteParams()
}

func (r *Route) call() {
	r.routeFunc(r.rq)
}
