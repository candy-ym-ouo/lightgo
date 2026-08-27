package render

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestTemplateExecutionErrorDoesNotWritePartialResponse(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "layout.html"), []byte(`{{template "content" .}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "broken.html"), []byte(`{{define "content"}}before {{.Missing}} after{{end}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	engine := NewTemplateEngine()
	if err := engine.Load(dir); err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	if err := engine.Render(response, 200, "broken", struct{ Name string }{Name: "test"}); err == nil {
		t.Fatal("expected template execution error")
	}
	if response.Body.Len() != 0 || response.Header().Get("Content-Type") != "" {
		t.Fatalf("partial response was written: headers=%v body=%q", response.Header(), response.Body.String())
	}
}
