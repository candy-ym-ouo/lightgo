package middlewares

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"lightgo/lightgo"
)

func TestRequestIDGzipAndRecovery(t *testing.T) {
	e := lightgo.New()
	e.Use(RequestID(), Recovery(nil), Gzip(gzip.BestSpeed))
	e.GET("/ok", func(c *lightgo.Context) { _ = c.Text(200, strings.Repeat("hello", 20)) })
	e.GET("/panic", func(c *lightgo.Context) { panic("boom") })
	e.GET("/length", func(c *lightgo.Context) {
		c.SetHeader("Content-Length", "100")
		_ = c.Text(200, strings.Repeat("x", 100))
	})
	r := httptest.NewRequest(http.MethodGet, "/ok", nil)
	r.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	e.ServeHTTP(w, r)
	if w.Header().Get("X-Request-ID") == "" || w.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("headers=%v", w.Header())
	}
	reader, err := gzip.NewReader(w.Body)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != strings.Repeat("hello", 20) {
		t.Fatalf("body=%q", body)
	}
	panicResponse := httptest.NewRecorder()
	panicRequest := httptest.NewRequest(http.MethodGet, "/panic", nil)
	panicRequest.Header.Set("Accept-Encoding", "gzip")
	e.ServeHTTP(panicResponse, panicRequest)
	if panicResponse.Code != 500 || panicResponse.Header().Get("Content-Encoding") != "" {
		t.Fatalf("panic response: code=%d headers=%v", panicResponse.Code, panicResponse.Header())
	}
	lengthResponse := httptest.NewRecorder()
	lengthRequest := httptest.NewRequest(http.MethodGet, "/length", nil)
	lengthRequest.Header.Set("Accept-Encoding", "gzip")
	e.ServeHTTP(lengthResponse, lengthRequest)
	if lengthResponse.Header().Get("Content-Length") != "" {
		t.Fatalf("compressed response retained Content-Length: %v", lengthResponse.Header())
	}
}

func TestCORSPreflight(t *testing.T) {
	e := lightgo.New()
	e.Use(CORS(CORSConfig{}))
	e.GET("/resource", func(c *lightgo.Context) { _ = c.StatusOnly(200) })
	r := httptest.NewRequest(http.MethodOptions, "/resource", nil)
	r.Header.Set("Origin", "https://example.test")
	w := httptest.NewRecorder()
	e.ServeHTTP(w, r)
	if w.Code != http.StatusNoContent || w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("response=%d %v", w.Code, w.Header())
	}
}

// [新增测试] 验证 Accept-Encoding 的 q 值和通配符协商不会误判。
func TestAcceptsGzipQuality(t *testing.T) {
	tests := []struct {
		header string
		want   bool
	}{
		{header: "gzip;q=0.5", want: true},
		{header: "br, gzip; q=0.5", want: true},
		{header: "gzip;q=0", want: false},
		{header: "gzip;q=0.0", want: false},
		{header: "*;q=0.5", want: true},
		{header: "br;q=1", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.header, func(t *testing.T) {
			if got := acceptsGzip(tt.header); got != tt.want {
				t.Fatalf("acceptsGzip(%q) = %v, want %v", tt.header, got, tt.want)
			}
		})
	}
}

// [新增测试] 验证 Gzip 外层、Recovery 内层时，写出响应后 panic 不会重复写入 500。
func TestGzipPreservesResponseStateForRecovery(t *testing.T) {
	e := lightgo.New()
	e.Use(Gzip(gzip.BestSpeed), Recovery(nil))
	e.GET("/panic-after-write", func(c *lightgo.Context) {
		_ = c.Text(http.StatusOK, "partial")
		panic("boom")
	})

	r := httptest.NewRequest(http.MethodGet, "/panic-after-write", nil)
	r.Header.Set("Accept-Encoding", "gzip;q=0.5")
	w := httptest.NewRecorder()
	e.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	reader, err := gzip.NewReader(w.Body)
	if err != nil {
		t.Fatalf("gzip.NewReader: %v", err)
	}
	body, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("io.ReadAll: %v", err)
	}
	if err := reader.Close(); err != nil {
		t.Fatalf("gzip reader close: %v", err)
	}
	if string(body) != "partial" {
		t.Fatalf("body = %q, want %q", body, "partial")
	}
}

// [新增测试] 验证 Gzip 包装后仍支持 Flusher，避免流式响应能力回归。
func TestGzipPreservesFlusher(t *testing.T) {
	e := lightgo.New()
	e.Use(Gzip(gzip.BestSpeed))
	e.GET("/stream", func(c *lightgo.Context) {
		flusher, ok := c.Writer.(http.Flusher)
		if !ok {
			t.Fatal("gzip response writer does not implement http.Flusher")
		}
		_, _ = c.Writer.Write([]byte("part"))
		flusher.Flush()
		_, _ = c.Writer.Write([]byte("ial"))
	})

	r := httptest.NewRequest(http.MethodGet, "/stream", nil)
	r.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	e.ServeHTTP(w, r)

	reader, err := gzip.NewReader(w.Body)
	if err != nil {
		t.Fatalf("gzip.NewReader: %v", err)
	}
	body, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("io.ReadAll: %v", err)
	}
	if string(body) != "partial" {
		t.Fatalf("body = %q, want %q", body, "partial")
	}
}
