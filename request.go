package superNova

import (
	"encoding/json"
	"errors"
	"github.com/valyala/fasthttp"
	"log"
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

func (r *Request) Send(data interface{}) {
	switch v := data.(type) {
	case []byte:
		r.Ctx.Write(v)
		break;
	case string:
		r.Ctx.Write([]byte(v))
		break;
	}
}

func (r *Request) SendJson(obj interface{}) error {
	json, err := json.Marshal(obj)
	if err != nil {
		log.Println(err)
		return err
	} else {
		r.Ctx.Response.Header.Set("Content-Type", "application/json")
		r.Ctx.Write(json)
	}
	return nil
}

