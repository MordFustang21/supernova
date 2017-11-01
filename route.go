package supernova

// Route is the construct of a single route pattern
type Route struct {
	routeFunc        Handler
	routeParamsIndex map[int]string
	route            string
}

// call builds the route params & executes the function tied to the route
func (r *Route) call(req *Request) {
	req.buildRouteParams(r.route)
	r.routeFunc(req)
}
