package supernova

import (
	"encoding/json"
	"errors"
	"github.com/valyala/fasthttp"
	"log"
	"strings"
	"context"
)

type Request struct {
	*fasthttp.RequestCtx
	RouteParams map[string]string
	Body        []byte
	BaseUrl     string
	ctx context.Context
}

func NewRequest(ctx *fasthttp.RequestCtx) *Request {
	request := Request{ctx, make(map[string]string), make([]byte, 0), "", context.Background()}
	request.Body = ctx.Request.Body()

	request.buildUrlParams()

	return &request
}

func (r *Request) Json(i interface{}) error {
	if r.Body == nil {
		return errors.New("Request Body is empty")
	} else {
		return json.Unmarshal(r.Body, i)
	}
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
	} else {
		r.Response.Header.Set("Content-Type", "application/json")
		r.Write(jsn)
	}
	return nil
}

func (r *Request) GetMethod() string {
	return string(r.Method())
}

// Builds url params and returns base route
func (r *Request) buildUrlParams() {
	reqUrl := string(r.Request.RequestURI())

	baseParts := strings.Split(reqUrl, "?")
	r.BaseUrl = baseParts[0]

	if len(baseParts) > 1 {
		params := strings.Join(baseParts[1:], "")

		paramParts := strings.Split(params, "&")

		for i := range paramParts {
			keyValue := strings.Split(paramParts[i], "=")
			if len(keyValue) > 1 {
				r.RouteParams[keyValue[0]] = keyValue[1]
			}
		}
	}
}
