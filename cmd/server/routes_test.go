package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"lightgo/internal/store"
	"lightgo/lightgo"
)

func TestSiteAndAuthenticatedPostFlow(t *testing.T) {
	e := lightgo.New()
	if err := registerRoutes(e, store.New(), filepath.Join("..", "..", "web")); err != nil {
		t.Fatal(err)
	}
	page := httptest.NewRecorder()
	e.ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/", nil))
	if page.Code != 200 || !strings.Contains(page.Body.String(), "LightGo") {
		t.Fatalf("page=%d %q", page.Code, page.Body.String())
	}

	login := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":"admin","password":"secret123"}`))
	req.Header.Set("Content-Type", "application/json")
	e.ServeHTTP(login, req)
	if login.Code != 200 {
		t.Fatalf("login=%d %s", login.Code, login.Body.String())
	}
	var envelope struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(login.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	create := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/blog/posts", strings.NewReader(`{"title":"Integration Post","summary":"summary","content":"content","category":"Test","status":"published"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+envelope.Data.Token)
	e.ServeHTTP(create, req)
	if create.Code != http.StatusCreated {
		t.Fatalf("create=%d %s", create.Code, create.Body.String())
	}

	categories := httptest.NewRecorder()
	e.ServeHTTP(categories, httptest.NewRequest(http.MethodGet, "/api/blog/categories", nil))
	if categories.Code != http.StatusOK || !strings.Contains(categories.Body.String(), "postCount") {
		t.Fatalf("categories=%d %s", categories.Code, categories.Body.String())
	}

	comment := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/blog/posts/1/comments", strings.NewReader(`{"content":"这是一条集成测试评论"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+envelope.Data.Token)
	e.ServeHTTP(comment, req)
	if comment.Code != http.StatusCreated {
		t.Fatalf("comment=%d %s", comment.Code, comment.Body.String())
	}

	comments := httptest.NewRecorder()
	e.ServeHTTP(comments, httptest.NewRequest(http.MethodGet, "/api/blog/posts/1/comments", nil))
	if comments.Code != http.StatusOK || !strings.Contains(comments.Body.String(), "集成测试评论") {
		t.Fatalf("comments=%d %s", comments.Code, comments.Body.String())
	}
}
