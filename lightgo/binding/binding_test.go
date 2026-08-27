package binding

import (
	"net/http/httptest"
	"strings"
	"testing"
)

type payload struct {
	Name   string   `json:"name" query:"name" validate:"required|min=3"`
	Age    int      `json:"age" query:"age" validate:"min=18"`
	Role   string   `json:"role" query:"role" default:"author" validate:"oneof=author admin"`
	Tags   []string `query:"tags"`
	Active bool     `query:"active"`
}

func TestJSONAndQueryBinding(t *testing.T) {
	r := httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"alice","age":20}`))
	r.Header.Set("Content-Type", "application/json")
	var got payload
	if err := Bind(r, &got, SourceAuto, nil); err != nil {
		t.Fatal(err)
	}
	if got.Name != "alice" || got.Age != 20 || got.Role != "author" {
		t.Fatalf("got=%+v", got)
	}

	r = httptest.NewRequest("GET", "/?name=bobby&age=21&tags=go,web&active=true", nil)
	got = payload{}
	if err := BindQuery(r, &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Tags) != 2 || !got.Active {
		t.Fatalf("got=%+v", got)
	}
}

func TestValidationErrors(t *testing.T) {
	r := httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"x","age":10,"role":"guest"}`))
	r.Header.Set("Content-Type", "application/json")
	var got payload
	err := BindJSON(r, &got)
	validation, ok := err.(ValidationErrors)
	if !ok || len(validation) != 3 {
		t.Fatalf("err=%T %v", err, err)
	}
}
