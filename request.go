package supernova

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/valyala/fasthttp"
	"golang.org/x/net/context"
)

// Request resembles an incoming request
type Request struct {
	*fasthttp.RequestCtx
	RouteParams map[string]string
	Body        []byte
	BaseUrl     string
	Ctx         context.Context
}

// buildRouteParams builds a map of the route params
func (r *Request) buildRouteParams(route string) {
	routeParams := r.RouteParams
	reqParts := strings.Split(r.BaseUrl, "/")
	routeParts := strings.Split(route[1:], "/")

	for index, val := range routeParts {
		if val[0] == ':' {
			routeParams[val[1:]] = reqParts[index]
		}
	}
}

// NewRequest creates a new Request pointer for an incoming request
func NewRequest(ctx *fasthttp.RequestCtx) *Request {
	request := Request{ctx, make(map[string]string), make([]byte, 0), "", context.Background()}
	request.Body = ctx.Request.Body()
	request.BaseUrl = string(request.URI().Path())

	return &request
}

// JSON unmarshals request body into the struct provided
func (r *Request) JSON(i interface{}) error {
	if r.Body == nil {
		return errors.New("Request Body is empty")
	}

	return json.Unmarshal(r.Body, i)
}

// Send writes the data to the response body
func (r *Request) Send(data interface{}) (int, error) {
	switch v := data.(type) {
	case []byte:
		return r.Write(v)
	case string:
		return r.Write([]byte(v))
	}
	return 0, errors.New("unsupported type")
}

// SendJSON converts any data type to JSON and attaches to the response body
func (r *Request) SendJSON(obj interface{}) (int, error) {
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
