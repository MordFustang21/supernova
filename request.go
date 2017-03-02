package supernova

import (
	"encoding/json"
	"errors"
	"log"
	"strings"

	"github.com/valyala/fasthttp"
	"golang.org/x/net/context"
)

type Request struct {
	*fasthttp.RequestCtx
	RouteParams map[string]string
	Body        []byte
	BaseUrl     string
	Ctx         context.Context
}

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

func NewRequest(ctx *fasthttp.RequestCtx) *Request {
	request := Request{ctx, make(map[string]string), make([]byte, 0), "", context.Background()}
	request.Body = ctx.Request.Body()
	request.BaseUrl = string(request.URI().Path())
	//request.buildUrlParams()

	return &request
}

func (r *Request) Json(i interface{}) error {
	if r.Body == nil {
		return errors.New("Request Body is empty")
	}
	
	return json.Unmarshal(r.Body, i)
}

func (r *Request) Send(data interface{}) {
	switch v := data.(type) {
	case []byte:
		r.Write(v)
		break
	case string:
		r.Write([]byte(v))
		break
	}
}

func (r *Request) SendJson(obj interface{}) error {
	jsn, err := json.Marshal(obj)
	if err != nil {
		log.Println(err)
		return err
	}

	r.Response.Header.Set("Content-Type", "application/json")
	r.Write(jsn)
	return nil
}

func (r *Request) GetMethod() string {
	return string(r.Method())
}

// Builds url params and returns base route
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
