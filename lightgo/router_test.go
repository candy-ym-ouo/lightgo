package lightgo

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestRouterPriorityParamsWildcardAndMethod(t *testing.T) {
	e := New()
	e.GET("/users/:id", func(c *Context) { _ = c.Text(200, "param:"+c.Param("id")) })
	e.GET("/users/new", func(c *Context) { _ = c.Text(200, "static") })
	e.GET("/assets/*path", func(c *Context) { _ = c.Text(200, c.Param("path")) })

	cases := []struct{ path, want string }{{"/users/new", "static"}, {"/users/42", "param:42"}, {"/assets/css/app.css", "css/app.css"}}
	for _, tc := range cases {
		r := httptest.NewRequest(http.MethodGet, tc.path, nil)
		w := httptest.NewRecorder()
		e.ServeHTTP(w, r)
		if w.Code != 200 || w.Body.String() != tc.want {
			t.Fatalf("%s: code=%d body=%q", tc.path, w.Code, w.Body.String())
		}
	}
	w := httptest.NewRecorder()
	e.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/users/42", nil))
	if w.Code != http.StatusMethodNotAllowed || w.Header().Get("Allow") != "GET" {
		t.Fatalf("405 response: %#v", w.Result())
	}
}

func TestMiddlewareOrderAndShortCircuit(t *testing.T) {
	e := New()
	var order []string
	e.Use(func(c *Context, next NextFunc) {
		order = append(order, "a-before")
		next()
		order = append(order, "a-after")
	})
	g := e.Group("/api", func(c *Context, next NextFunc) {
		order = append(order, "b-before")
		next()
		order = append(order, "b-after")
	})
	g.GET("/ok", func(c *Context) { order = append(order, "handler"); _ = c.StatusOnly(204) })
	w := httptest.NewRecorder()
	e.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/ok", nil))
	want := []string{"a-before", "b-before", "handler", "b-after", "a-after"}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("order=%v want=%v", order, want)
	}

	e2 := New()
	e2.Use(func(c *Context, next NextFunc) { _ = c.Text(401, "stop") })
	e2.GET("/", func(c *Context) { t.Fatal("handler should not run") })
	w = httptest.NewRecorder()
	e2.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != 401 {
		t.Fatalf("code=%d", w.Code)
	}
}
