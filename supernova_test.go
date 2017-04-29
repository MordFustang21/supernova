package supernova

import (
	"bytes"
	"fmt"
	"net"
	"testing"
	"time"
)

// Test adding Routes
func TestServer_All(t *testing.T) {
	urlHit := false
	s := New()
	s.All("/test", func(r *Request) {
		urlHit = true
	})

	if s.paths[""].children["test"] == nil {
		t.Error("Failed to insert all route")
	}

	err := sendRequest(s, "OPTION", "/test")
	if err != nil {
		t.Error(err)
	}

	if !urlHit {
		t.Error("All Url not hit")
	}
}

func TestServer_Get(t *testing.T) {
	urlHit := false

	s := New()
	s.Get("/test", func(r *Request) {
		urlHit = true
	})

	if s.paths["GET"].children["test"] == nil {
		t.Error("Failed to insert GET route")
	}

	err := sendRequest(s, "GET", "/test")
	if err != nil {
		t.Error(err)
	}

	if !urlHit {
		t.Error("All Url not hit")
	}
}

func TestServer_Put(t *testing.T) {
	urlHit := false

	s := New()
	s.Put("/test", func(r *Request) {
		urlHit = true
	})

	if s.paths["PUT"].children["test"] == nil {
		t.Error("Failed to insert PUT route")
	}

	err := sendRequest(s, "PUT", "/test")
	if err != nil {
		t.Error(err)
	}

	if !urlHit {
		t.Error("All Url not hit")
	}
}

func TestServer_Post(t *testing.T) {
	urlHit := false

	s := New()
	s.Post("/test", func(r *Request) {
		urlHit = true
	})

	if s.paths["POST"].children["test"] == nil {
		t.Error("Failed to insert POST route")
	}

	err := sendRequest(s, "POST", "/test")
	if err != nil {
		t.Error(err)
	}

	if !urlHit {
		t.Error("All Url not hit")
	}

}
func TestServer_Delete(t *testing.T) {
	urlHit := false

	s := New()
	s.Delete("/test", func(r *Request) {
		urlHit = true
	})

	if s.paths["DELETE"].children["test"] == nil {
		t.Error("Failed to insert DELETE route")
	}

	err := sendRequest(s, "DELETE", "/test")
	if err != nil {
		t.Error(err)
	}

	if !urlHit {
		t.Error("All Url not hit")
	}
}

// Check middleware
func TestServer_Use(t *testing.T) {
	s := New()
	s.Use(func(r *Request, next func()) {

	})

	if len(s.middleWare) != 1 {
		t.Error("Middlware wasn't added")
	}
}

func TestServer_Restricted(t *testing.T) {
	urlHit := false

	s := New()
	s.Restricted("OPTION", "/test", func(*Request) {
		urlHit = true
	})

	if s.paths["OPTION"].children["test"] == nil {
		t.Error("Route wasn't restricted to method")
	}

	err := sendRequest(s, "OPTION", "/test")
	if err != nil {
		t.Error(err)
	}

	if !urlHit {
		t.Error("All Url not hit")
	}
}

func TestMultipleChildren(t *testing.T) {
	s := New()
	s.All("/test/stuff", func(*Request) {

	})

	s.All("/test/test", func(*Request) {

	})

	if len(s.paths[""].children["test"].children) != 2 {
		t.Error("Node possibly overwritten")
	}
}

// Test finding Routes
func TestServer_climbTree(t *testing.T) {
	cases := []struct {
		Method    string
		Path      string
		ExpectNil bool
	}{
		{
			"GET",
			"/test",
			false,
		},
		{
			"GET",
			"/stuff/param1/params/param2",
			false,
		},
		{
			"GET",
			"/stuff/param1/par/param2",
			true,
		},
	}

	s := New()
	s.Get("/test", func(*Request) {

	})

	s.Get("/stuff/:test/params/:more", func(*Request) {

	})

	for _, val := range cases {
		node := s.climbTree(val.Method, val.Path)
		if val.ExpectNil {
			if node != nil {
				t.Errorf("%s Expected nil got *Node", val.Path)
			}
		} else {
			if node == nil {
				t.Errorf("%s Expected *Node got nil", val.Path)
			}
		}
	}
}

func TestServer_EnableGzip(t *testing.T) {
	s := New()
	s.EnableGzip(true)

	if !s.compressionEnabled {
		t.Error("EnableGzip wasn't set")
	}
}

func TestServer_EnableDebug(t *testing.T) {
	s := New()
	s.EnableDebug(true)

	if !s.debug {
		t.Error("Debug mode wasn't set")
	}
}

func TestNew(t *testing.T) {
	s := New()
	if s == nil {
		t.Error("Expected *Server got nil")
	}
}

func TestServer_SetShutDownHandler(t *testing.T) {
	s := New()
	s.SetShutDownHandler(func() {

	})

	if s.shutdownHandler == nil {
		t.Error("Shutdown handler not set")
	}
}

func sendRequest(s *Server, method, url string) error {
	rw := &readWriter{}
	rw.r.WriteString(fmt.Sprintf("%s %s HTTP/1.1\r\n\r\n", method, url))

	err := s.server.ServeConn(rw)
	if err != nil {
		return err
	}

	return nil
}

type readWriter struct {
	net.Conn
	r bytes.Buffer
	w bytes.Buffer
}

var zeroTCPAddr = &net.TCPAddr{
	IP: net.IPv4zero,
}

func (rw *readWriter) Close() error {
	return nil
}

func (rw *readWriter) Read(b []byte) (int, error) {
	return rw.r.Read(b)
}

func (rw *readWriter) Write(b []byte) (int, error) {
	return rw.w.Write(b)
}

func (rw *readWriter) RemoteAddr() net.Addr {
	return zeroTCPAddr
}

func (rw *readWriter) LocalAddr() net.Addr {
	return zeroTCPAddr
}

func (rw *readWriter) SetReadDeadline(t time.Time) error {
	return nil
}

func (rw *readWriter) SetWriteDeadline(t time.Time) error {
	return nil
}
