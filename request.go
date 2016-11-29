package supernova

import (
	"encoding/json"
	"errors"
	"github.com/valyala/fasthttp"
	"log"
)

type Request struct {
	*fasthttp.RequestCtx
	RouteParams map[string]string
	Body        []byte
}

func NewRequest(ctx *fasthttp.RequestCtx) *Request {
	request := Request{ctx, make(map[string]string), make([]byte, 0)}
	request.Body = ctx.Request.Body()

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
		break;
	case string:
		r.Write([]byte(v))
		break;
	}
}

func (r *Request) SendJson(obj interface{}) error {
	json, err := json.Marshal(obj)
	if err != nil {
		log.Println(err)
		return err
	} else {
		r.Response.Header.Set("Content-Type", "application/json")
		r.Write(json)
	}
	return nil
}

func (r *Request) GetMethod() string {
	return string(r.Method())
}
