package settings

import (
	"embed"
	"net/http"
	"path"
)

//go:embed page
var pageFiles embed.FS

// The types are written down rather than looked up: Go reads them from the Windows registry,
// and a wrong entry there makes the browser refuse the file under nosniff.
var pageTypes = map[string]string{
	".html": "text/html; charset=utf-8",
	".js":   "text/javascript; charset=utf-8",
	".css":  "text/css; charset=utf-8",
}

func pageFile(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		body, err := pageFiles.ReadFile("page/" + name)
		if err != nil {
			failed(w, err)
			return
		}

		w.Header().Set("Content-Type", pageTypes[path.Ext(name)])
		w.Write(body)
	}
}
