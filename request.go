package supernova

import (
	"encoding/json"
	"errors"
	"io"
	"strings"

	"github.com/valyala/fasthttp"
	"golang.org/x/net/context"
)

// Request resembles an incoming request
type Request struct {
	*fasthttp.RequestCtx
	RouteParams map[string]string
	BaseUrl     string

	// Writer is used to write to response body
	Writer io.Writer
	Ctx    context.Context
}

// buildRouteParams builds a map of the route params
func (r *Request) buildRouteParams(route string) {
	routeParams := r.RouteParams
	reqParts := strings.Split(r.BaseUrl[1:], "/")
	routeParts := strings.Split(route[1:], "/")

	for index, val := range routeParts {
		if val[0] == ':' {
			println(val[1:] + reqParts[index])
			routeParams[val[1:]] = reqParts[index]
		}
	}
}

// NewRequest creates a new Request pointer for an incoming request
func NewRequest(ctx *fasthttp.RequestCtx) *Request {
	req := new(Request)
	req.RequestCtx = ctx
	req.RouteParams = make(map[string]string)
	req.BaseUrl = string(ctx.URI().Path())
	req.Writer = ctx.Response.BodyWriter()

	return req
}

// JSON unmarshals request body into the struct provided
func (r *Request) ReadJSON(i interface{}) error {
	//TODO: detect body size and use reader if necessary
	return json.Unmarshal(r.Request.Body(), i)
}

// Send writes the data to the response body
func (r *Request) Send(data interface{}) (int, error) {
	switch v := data.(type) {
	case []byte:
		return r.Write(v)
	case string:
		return r.Write([]byte(v))
	case error:
		return r.Write([]byte(v.Error()))
	}

	return 0, errors.New("unsupported type")
}

// JSON marshals the given interface object and writes the JSON response.
func (r *Request) JSON(obj interface{}) (int, error) {
	jsn, err := json.Marshal(obj)
	if err != nil {
		return 0, err
	}

	r.Response.Header.Set("Content-Type", "application/json")
	return r.Write(jsn)
}

// GetMethod provides a simple way to return the request method type as a string
func (r *Request) GetMethod() string {
	return string(r.Method())
}

// buildUrlParams builds url params and returns base route
func (r *Request) buildUrlParams() {
	reqUrl := string(r.Request.RequestURI())
	baseParts := strings.Split(reqUrl, "?")

	if len(baseParts) == 0 {
		return
	}

	params := strings.Join(baseParts[1:], "")

	paramParts := strings.Split(params, "&")

	for i := range paramParts {
		keyValue := strings.Split(paramParts[i], "=")
		if len(keyValue) > 1 {
			r.RouteParams[keyValue[0]] = keyValue[1]
		}
	}
}
