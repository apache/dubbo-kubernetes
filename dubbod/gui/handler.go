package gui

import (
	"bytes"
	"encoding/json"
	"html/template"
	"net/http"
	"path"
	"strings"

	"github.com/apache/dubbo-kubernetes/dubbod/gui/resources"
)

type Config struct {
	BasePath string `json:"basePath"`
	Product  string `json:"product"`
}

type templateData struct {
	BasePath   string
	ConfigJSON template.JS
}

func NewHandler(basePath string, cfg Config) (http.Handler, error) {
	basePath = NormalizeBasePath(basePath)
	cfg.BasePath = basePath

	payload, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	guiFS := resources.FS()
	tmpl, err := template.ParseFS(guiFS, "index.html")
	if err != nil {
		return nil, err
	}

	buf := bytes.Buffer{}
	if err := tmpl.Execute(&buf, templateData{
		BasePath:   basePath,
		ConfigJSON: template.JS(payload),
	}); err != nil {
		return nil, err
	}

	prefix := basePath
	if prefix != "/" {
		prefix += "/"
	}
	return http.StripPrefix(prefix, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		name := strings.TrimPrefix(path.Clean("/"+request.URL.Path), "/")
		if name != "" && name != "." {
			if _, err := guiFS.Open(name); err == nil {
				http.FileServer(http.FS(guiFS)).ServeHTTP(writer, request)
				return
			}
		}

		writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write(buf.Bytes())
	})), nil
}

func NormalizeBasePath(basePath string) string {
	trimmed := strings.TrimSpace(basePath)
	if trimmed == "" || trimmed == "/" {
		return "/"
	}

	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}

	return strings.TrimRight(trimmed, "/")
}
