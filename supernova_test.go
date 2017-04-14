package supernova

import (
	"testing"
)

// Test adding Routes
func TestServer_All(t *testing.T) {
	s := New()
	s.All("/test", func(r *Request) {

	})

	if s.paths[""].children["test"] == nil {
		t.Error("Failed to insert all route")
	}
}

func TestServer_Get(t *testing.T) {
	s := New()
	s.Get("/test", func(r *Request) {

	})

	if s.paths["GET"].children["test"] == nil {
		t.Error("Failed to insert GET route")
	}
}

func TestServer_Put(t *testing.T) {
	s := New()
	s.Put("/test", func(r *Request) {

	})

	if s.paths["PUT"].children["test"] == nil {
		t.Error("Failed to insert PUT route")
	}
}

func TestServer_Post(t *testing.T) {
	s := New()
	s.Post("/test", func(r *Request) {

	})

	if s.paths["POST"].children["test"] == nil {
		t.Error("Failed to insert POST route")
	}
}
func TestServer_Delete(t *testing.T) {
	s := New()
	s.Delete("/test", func(r *Request) {

	})

	if s.paths["DELETE"].children["test"] == nil {
		t.Error("Failed to insert DELETE route")
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
	s := New()
	s.Restricted("OPTION", "/test", func(*Request) {

	})

	if s.paths["OPTION"].children["test"] == nil {
		t.Error("Route wasn't restricted to method")
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
