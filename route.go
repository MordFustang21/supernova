package supernova

type Route struct {
	routeFunc        func(*Request)
	routeParamsIndex map[int]string
	route            string
}

func (r *Route) call(req *Request) {
	req.buildRouteParams(r.route)
	r.routeFunc(req)
}
