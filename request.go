package superNova

import (
	"encoding/json"
	"errors"
	"github.com/valyala/fasthttp"
)

type Request struct {
	Ctx         *fasthttp.RequestCtx
	RouteParams map[string]string
	Body        []byte
}

func NewRequest(ctx *fasthttp.RequestCtx) *Request {
	request := new(Request)
	request.Ctx = ctx
	request.Body = ctx.Request.Body()

	return request
}

func (r *Request) Json(i interface{}) error {
	if r.Body == nil {
		return errors.New("Request Body is empty")
	} else {
		return json.Unmarshal(r.Body, i)
	}
}

