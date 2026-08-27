package middlewares

import (
	"compress/gzip"
	"lightgo/lightgo"
	"net/http"
	"strconv"
	"strings"
)

type gzipWriter struct {
	http.ResponseWriter
	writer     *gzip.Writer
	status     int
	size       int
	compressed bool
}

func (w *gzipWriter) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
}
func (w *gzipWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	if !w.compressed {
		w.Header().Del("Content-Length")
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Add("Vary", "Accept-Encoding")
		w.ResponseWriter.WriteHeader(w.status)
		w.compressed = true
	}
	n, err := w.writer.Write(data)
	w.size += n
	return n, err
}

// [修改] 暴露包装后的响应状态，确保 Context 在 Gzip 中间件下仍能判断响应是否已提交。
func (w *gzipWriter) Status() int { return w.status }
func (w *gzipWriter) Size() int   { return w.size }

// [修改] 保留原 responseWriter 的流式响应能力。
func (w *gzipWriter) Flush() {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	if !w.compressed {
		w.Header().Del("Content-Length")
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Add("Vary", "Accept-Encoding")
		w.ResponseWriter.WriteHeader(w.status)
		w.compressed = true
	}
	_ = w.writer.Flush()
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
func (w *gzipWriter) Close() error {
	if w.compressed {
		return w.writer.Close()
	}
	if w.status != 0 {
		w.ResponseWriter.WriteHeader(w.status)
	}
	return nil
}
func Gzip(level int) lightgo.Middleware {
	if level < gzip.HuffmanOnly || level > gzip.BestCompression {
		level = gzip.DefaultCompression
	}
	return func(c *lightgo.Context, next lightgo.NextFunc) {
		if !acceptsGzip(c.Header("Accept-Encoding")) || c.Request.Method == http.MethodHead {
			next()
			return
		}
		writer, _ := gzip.NewWriterLevel(c.Writer, level)
		original := c.Writer
		wrapped := &gzipWriter{ResponseWriter: original, writer: writer}
		c.Writer = wrapped
		defer func() {
			_ = wrapped.Close()
			c.Writer = original
		}()
		next()
	}
}
func acceptsGzip(header string) bool {
	var wildcardQuality *float64
	for _, token := range strings.Split(header, ",") {
		parts := strings.Split(token, ";")
		coding := strings.TrimSpace(parts[0])
		quality := 1.0
		for _, parameter := range parts[1:] {
			name, value, ok := strings.Cut(strings.TrimSpace(parameter), "=")
			if !ok || !strings.EqualFold(strings.TrimSpace(name), "q") {
				continue
			}
			parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
			if err != nil || parsed < 0 || parsed > 1 {
				quality = 0
			} else {
				quality = parsed
			}
			break
		}
		switch {
		case strings.EqualFold(coding, "gzip"):
			// [修改] 解析 q 值，避免把 q=0.5 误判为 q=0。
			return quality > 0
		case coding == "*":
			wildcardQuality = &quality
		}
	}
	return wildcardQuality != nil && *wildcardQuality > 0
}
