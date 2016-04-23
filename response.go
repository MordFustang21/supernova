package superNova

import (
	"net/http"
	"encoding/json"
	"log"
)

type Response struct {
	R http.ResponseWriter
}

func (r *Response) Send(data interface{}) {
	switch v := data.(type) {
	case []byte:
		r.R.Write(v)
		break;
	case string:
		r.R.Write([]byte(v))
		break;
	}
}

func (r *Response) Json(obj interface{}) error {
	json, err := json.Marshal(obj)
	if err != nil {
		log.Println(err)
		return err
	} else {
		r.R.Header().Set("Content-Type", "application/json")
		r.R.Write(json)
	}
	return nil
}
