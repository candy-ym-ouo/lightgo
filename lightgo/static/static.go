package static

import (
	"crypto/sha256"
	"fmt"
	"lightgo/lightgo"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Options struct {
	CacheControl string
	Index        string
}

func FileServer(dir string, options Options) lightgo.HandlerFunc {
	root, err := filepath.Abs(dir)
	if err != nil {
		panic(err)
	}
	if options.CacheControl == "" {
		options.CacheControl = "public, max-age=3600"
	}
	if options.Index == "" {
		options.Index = "index.html"
	}
	return func(c *lightgo.Context) {
		raw := c.Param("path")
		clean := filepath.Clean("/" + raw)
		candidate := filepath.Join(root, filepath.FromSlash(strings.TrimPrefix(clean, "/")))
		absolute, err := filepath.Abs(candidate)
		if err != nil || !inside(root, absolute) {
			_ = c.Error(lightgo.NewHTTPError(http.StatusForbidden, "禁止访问"))
			return
		}
		info, err := os.Stat(absolute)
		if err == nil && info.IsDir() {
			absolute = filepath.Join(absolute, options.Index)
			info, err = os.Stat(absolute)
		}
		if err != nil || info.IsDir() {
			_ = c.Error(lightgo.NewHTTPError(http.StatusNotFound, "静态资源不存在"))
			return
		}
		etag := fmt.Sprintf(`"%x"`, sha256.Sum256([]byte(fmt.Sprintf("%s:%d:%d", absolute, info.Size(), info.ModTime().UnixNano()))))
		if c.Request.Header.Get("If-None-Match") == etag {
			c.SetHeader("ETag", etag)
			_ = c.StatusOnly(http.StatusNotModified)
			return
		}
		if modified := c.Request.Header.Get("If-Modified-Since"); modified != "" {
			if since, parseErr := http.ParseTime(modified); parseErr == nil && !info.ModTime().After(since.Add(timeSecond)) {
				_ = c.StatusOnly(http.StatusNotModified)
				return
			}
		}
		contentType := mime.TypeByExtension(filepath.Ext(absolute))
		if contentType != "" {
			c.SetHeader("Content-Type", contentType)
		}
		c.SetHeader("Cache-Control", options.CacheControl)
		c.SetHeader("ETag", etag)
		c.SetHeader("Last-Modified", info.ModTime().UTC().Format(http.TimeFormat))
		if err := c.File(absolute, false); err != nil {
			_ = c.Error(lightgo.NewHTTPError(500, "读取静态资源失败", err))
		}
	}
}

const timeSecond = 1_000_000_000

func inside(root, candidate string) bool {
	rel, err := filepath.Rel(root, candidate)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
