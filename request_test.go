package supernova

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/valyala/fasthttp"
)

func TestNewRequest(t *testing.T) {
	r := NewRequest(new(fasthttp.RequestCtx))
	if r == nil {
		t.Error("Request is nil")
	}
}

func TestRequest_Error(t *testing.T) {
	r := NewRequest(new(fasthttp.RequestCtx))
	r.Error(500, "Internal Error")

	b := string(r.Response.Body())
	con := strings.Contains(b, ":\"Internal Error\"}")

	if r.Response.StatusCode() != 500 {
		t.Error("Incorrect status code")
	}

	if !con {
		t.Error("Response didn't contain json")
	}
}

type jStruct struct {
	Key string `json:"key"`
}

func TestRequest_JSON(t *testing.T) {
	r := NewRequest(new(fasthttp.RequestCtx))
	jsn := jStruct{Key: "Test Key"}

	r.JSON(200, jsn)

	var jsonResponse jStruct
	err := json.Unmarshal(r.Response.Body(), &jsonResponse)
	if err != nil {
		t.Error(err)
	}

	if jsonResponse.Key != "Test Key" {
		t.Error("Didn't write body correctly")
	}

}

func TestRequest_ReadJSON(t *testing.T) {
	r := NewRequest(new(fasthttp.RequestCtx))
	r.Request.SwapBody([]byte("{\"key\":\"test\"}"))

	var jsn jStruct
	err := r.ReadJSON(&jsn)
	if err != nil {
		t.Error(err)
	}

	if jsn.Key != "test" {
		t.Error("Didn't get test on key")
	}
}

func TestRequest_Param(t *testing.T) {
	s := New()

	s.Get("/user/:name", func(r *Request) {
		n := r.Param("name")
		if n != "gopher" {
			t.Errorf("Expected gopher got %s", n)
		}
	})

	err := sendRequest(s, "GET", "/user/gopher")
	if err != nil {
		t.Error(err)
	}
}
