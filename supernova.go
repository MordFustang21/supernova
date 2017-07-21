// Package supernova is a fasthttp router that implements a radix tree for fast lookups
package supernova

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/valyala/fasthttp"
)

// Server represents the router and all associated data
type Server struct {
	server *fasthttp.Server
	ln     net.Listener

	// radix tree for looking up routes
	paths      map[string]*Node
	middleWare []Middleware

	// shutdown function called when ctl-c is intercepted
	shutdownHandler func()

	// debug defines logging for requests
	debug bool
}

// Node holds a single route with accompanying children routes
type Node struct {
	route    *Route
	isEdge   bool
	children map[string]*Node
}

// CachedObj represents a static asset
type CachedObj struct {
	data       []byte
	timeCached time.Time
}

// CachedStatic holds all cached static assets in memory
type CachedStatic struct {
	mutex sync.Mutex
	files map[string]*CachedObj
}

// Middleware holds all middleware functions
type Middleware struct {
	middleFunc func(*Request, func())
}

// New returns new supernova router
func New() *Server {
	s := new(Server)

	s.server = &fasthttp.Server{
		Handler: s.handler,
	}

	return s
}

// EnableDebug toggles output for incoming requests
func (sn *Server) EnableDebug(debug bool) {
	if debug {
		sn.debug = true
	}
}

// ListenAndServe starts the server
func (sn *Server) ListenAndServe(addr string) error {
	listener, err := net.Listen("tcp4", addr)
	if err != nil {
		return err
	}

	sn.ln = NewGracefulListener(listener, time.Second*5)
	return sn.server.Serve(sn.ln)
}

// ServeTLS starts server with ssl
func (sn *Server) ListenAndServeTLS(addr, certFile, keyFile string) error {
	listener, err := net.Listen("tcp4", addr)
	if err != nil {
		return err
	}

	sn.ln = NewGracefulListener(listener, time.Second*5)
	return fasthttp.ListenAndServeTLS(addr, certFile, keyFile, sn.handler)
}

// Serve serves incoming connections from the given listener.
//
// Serve blocks until the given listener returns permanent error.
func (sn *Server) Serve(ln net.Listener) error {
	return sn.server.Serve(ln)
}

// Close closes existing listener
func (sn *Server) Close() error {
	return sn.ln.Close()
}

// handler is the main entry point into the router
func (sn *Server) handler(ctx *fasthttp.RequestCtx) {
	request := NewRequest(ctx)
	var logMethod func()
	if sn.debug {
		logMethod = getDebugMethod(request)
	}

	if logMethod != nil {
		defer logMethod()
	}

	// Run Middleware
	finished := sn.runMiddleware(request)
	if !finished {
		return
	}

	route := sn.climbTree(request.GetMethod(), request.BaseUrl)
	if route != nil {
		route.call(request)
		return
	}

	ctx.Error("404 Not Found", fasthttp.StatusNotFound)
}

// All adds route for all http methods
func (sn *Server) All(route string, routeFunc func(*Request)) {
	sn.addRoute("", buildRoute(route, routeFunc))
}

// Get adds only GET method to route
func (sn *Server) Get(route string, routeFunc func(*Request)) {
	sn.addRoute("GET", buildRoute(route, routeFunc))
}

// Post adds only POST method to route
func (sn *Server) Post(route string, routeFunc func(*Request)) {
	sn.addRoute("POST", buildRoute(route, routeFunc))
}

// Put adds only PUT method to route
func (sn *Server) Put(route string, routeFunc func(*Request)) {
	sn.addRoute("PUT", buildRoute(route, routeFunc))
}

// Delete adds only DELETE method to route
func (sn *Server) Delete(route string, routeFunc func(*Request)) {
	sn.addRoute("DELETE", buildRoute(route, routeFunc))
}

// Restricted adds route that is restricted by method
func (sn *Server) Restricted(method, route string, routeFunc func(*Request)) {
	sn.addRoute(method, buildRoute(route, routeFunc))
}

// addRoute takes route and method and adds it to route tree
func (sn *Server) addRoute(method string, route *Route) {
	routeStr := route.route
	if routeStr[len(routeStr)-1] == '/' {
		routeStr = routeStr[:len(routeStr)-1]
		route.route = routeStr
	}
	if sn.paths == nil {
		sn.paths = make(map[string]*Node)
	}

	if sn.paths[method] == nil {
		node := new(Node)
		node.children = make(map[string]*Node)
		sn.paths[method] = node
	}

	parts := strings.Split(routeStr[1:], "/")

	currentNode := sn.paths[method]
	for index, val := range parts {
		childKey := val
		if val[0] == ':' {
			childKey = ""
		} else {
			childKey = val
		}

		if node, ok := currentNode.children[childKey]; ok {
			currentNode = node
		} else {
			node := getNode(false, nil)
			currentNode.children[childKey] = node
			currentNode = node
		}

		if index == len(parts)-1 {
			node := getNode(true, route)
			currentNode.children[childKey] = node
			currentNode = node
		}
	}
}

// getNode builds a new node to be added to the radix tree
func getNode(isEdge bool, route *Route) *Node {
	node := new(Node)
	node.children = make(map[string]*Node)
	if isEdge {
		node.isEdge = true
		node.route = route
	}
	return node
}

// climbTree takes in path and traverses tree to find route
func (sn *Server) climbTree(method, path string) *Route {
	// strip slashes
	if path[len(path)-1] == '/' {
		path = path[1 : len(path)-1]
	} else {
		path = path[1:]
	}

	parts := strings.Split(path, "/")
	pathLen := len(parts) - 1

	currentNode, ok := sn.paths[method]
	if !ok {
		currentNode, ok = sn.paths[""]
		if !ok {
			return nil
		}
	}

	for index, val := range parts {
		var node *Node

		node = currentNode.children[val]
		if node == nil {
			node = currentNode.children[""]
		}

		// path not found return
		if node == nil && method == "" {
			return nil
		} else if node == nil {
			return sn.climbTree("", path)
		}

		currentNode = node

		// if at end return current route
		if index == pathLen {
			if node, ok := currentNode.children[val]; ok {
				return node.route
			}

			if node, ok = currentNode.children[""]; ok {
				return node.route
			}

		}
	}

	return nil
}

// buildRoute creates new Route
func buildRoute(route string, routeFunc func(*Request)) *Route {
	routeObj := new(Route)
	routeObj.routeFunc = routeFunc
	routeObj.routeParamsIndex = make(map[int]string)
	routeObj.route = route

	return routeObj
}

// Use adds a new function to the middleware stack
func (sn *Server) Use(f func(*Request, func())) {
	if sn.middleWare == nil {
		sn.middleWare = make([]Middleware, 0)
	}
	middle := new(Middleware)
	middle.middleFunc = f
	sn.middleWare = append(sn.middleWare, *middle)
}

// Internal method that runs the middleware
func (sn *Server) runMiddleware(req *Request) bool {
	stackFinished := true
	for m := range sn.middleWare {
		nextCalled := false
		sn.middleWare[m].middleFunc(req, func() {
			nextCalled = true
		})

		if !nextCalled {
			stackFinished = false
			break
		}
	}

	return stackFinished
}

// SetShutDownHandler implements function called when SIGTERM signal is received
func (sn *Server) SetShutDownHandler(shutdownFunc func()) {
	sn.shutdownHandler = shutdownFunc

	sigs := make(chan os.Signal)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for {
			select {
			case <-sigs:
				err := sn.ln.Close()
				if err != nil {
					fmt.Printf("Error closing conn: %s\n", err.Error())
				}

				if shutdownFunc != nil {
					shutdownFunc()
				}
				os.Exit(0)
			}
		}
	}()
}
