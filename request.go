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

	routeParams map[string]string
	queryParams map[string]string
	BaseUrl     string

	// Writer is used to write to response body
	Writer io.Writer
	Ctx    context.Context
}

// JSONError resembles the RESTful standard for an error response
type JSONError struct {
	Errors  []interface{} `json:"errors"`
	Code    int           `json:"code"`
	Message string        `json:"message"`
}

// JSONErrors holds the JSONError response
type JSONErrors struct {
	Error JSONError `json:"error"`
}

// NewRequest creates a new Request pointer for an incoming request
func NewRequest(ctx *fasthttp.RequestCtx) *Request {
	req := new(Request)
	req.RequestCtx = ctx
	req.routeParams = make(map[string]string)
	req.queryParams = make(map[string]string)
	req.BaseUrl = string(ctx.URI().Path())
	req.Writer = ctx.Response.BodyWriter()

	return req
}

// RouteParam checks for and returns param or "" if doesn't exist
func (r *Request) RouteParam(key string) string {
	if val, ok := r.routeParams[key]; ok {
		return val
	}

	return ""
}

// QueryParam checks for and returns param or "" if doesn't exist
func (r *Request) QueryParam(key string) string {
	if val, ok := r.queryParams[key]; ok {
		return val
	}

	return ""
}

// Error allows an easy method to set the RESTful standard error response
func (r *Request) Error(statusCode int, msg string, errors ...interface{}) (int, error) {
	r.Response.Reset()
	newErr := JSONErrors{
		Error: JSONError{
			Errors:  errors,
			Code:    statusCode,
			Message: msg,
		},
	}
	return r.JSON(statusCode, newErr)
}

// buildRouteParams builds a map of the route params
func (r *Request) buildRouteParams(route string) {
	routeParams := r.routeParams
	reqParts := strings.Split(r.BaseUrl[1:], "/")
	routeParts := strings.Split(route[1:], "/")

	for index, val := range routeParts {
		if val[0] == ':' {
			routeParams[val[1:]] = reqParts[index]
		}
	}
}

// buildQueryParams parses out all query params and places them in map
func (r *Request) buildQueryParams() {
	r.RequestCtx.QueryArgs().VisitAll(func(key, value []byte) {
		r.queryParams[string(key)] = string(value)
	})
}

// ReadJSON unmarshals request body into the struct provided
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
func (r *Request) JSON(code int, obj interface{}) (int, error) {
	jsn, err := json.Marshal(obj)
	if err != nil {
		return 0, err
	}

	r.Response.Header.Set("Content-Type", "application/json")
	r.SetStatusCode(code)
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
			r.routeParams[keyValue[0]] = keyValue[1]
		}
	}
}
