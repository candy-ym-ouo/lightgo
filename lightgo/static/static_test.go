package static

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"lightgo/lightgo"
)

func TestFileServerCacheAndMissing(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	e := lightgo.New()
	e.GET("/static/*path", FileServer(dir, Options{}))
	w := httptest.NewRecorder()
	e.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/static/hello.txt", nil))
	if w.Code != 200 || w.Body.String() != "hello" || w.Header().Get("ETag") == "" {
		t.Fatalf("response=%d %q %#v", w.Code, w.Body.String(), w.Header())
	}
	r := httptest.NewRequest(http.MethodGet, "/static/hello.txt", nil)
	r.Header.Set("If-None-Match", w.Header().Get("ETag"))
	cached := httptest.NewRecorder()
	e.ServeHTTP(cached, r)
	if cached.Code != http.StatusNotModified {
		t.Fatalf("cached code=%d", cached.Code)
	}
	missing := httptest.NewRecorder()
	e.ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "/static/nope", nil))
	if missing.Code != 404 {
		t.Fatalf("missing code=%d", missing.Code)
	}
}
