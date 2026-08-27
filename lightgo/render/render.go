package render

import (
	"encoding/json"
	"encoding/xml"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
)

func JSON(w http.ResponseWriter, status int, value any) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(value)
}
func XML(w http.ResponseWriter, status int, value any) error {
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(status)
	_, err := w.Write([]byte(xml.Header))
	if err != nil {
		return err
	}
	return xml.NewEncoder(w).Encode(value)
}
func Text(w http.ResponseWriter, status int, value string) error {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, err := io.WriteString(w, value)
	return err
}
func Blob(w http.ResponseWriter, status int, contentType string, data []byte) error {
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(status)
	_, err := w.Write(data)
	return err
}
func File(w http.ResponseWriter, r *http.Request, filename string, download bool) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	contentType := mime.TypeByExtension(filepath.Ext(filename))
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	if download {
		w.Header().Set("Content-Disposition", `attachment; filename="`+filepath.Base(filename)+`"`)
	}
	http.ServeContent(w, r, info.Name(), info.ModTime(), file)
	return nil
}
func Redirect(w http.ResponseWriter, r *http.Request, status int, location string) error {
	http.Redirect(w, r, location, status)
	return nil
}
func Status(w http.ResponseWriter, status int) error {
	w.WriteHeader(status)
	return nil
}
