package web

import (
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func (server *Server) serveStatic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	w.Header().Set("Cache-Control", "no-store")

	name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
	if name == "." || name == "" {
		name = "static/index.html"
	}
	if !strings.HasPrefix(name, "static/") {
		http.NotFound(w, r)
		return
	}
	content, err := server.readStaticFile(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if contentType := mime.TypeByExtension(path.Ext(name)); contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	if r.Method == http.MethodHead {
		return
	}
	_, _ = w.Write(content)
}

// readStaticFile returns a static asset, reading from disk when staticDir is
// configured (dev mode) and falling back to the embedded copy otherwise.
func (server *Server) readStaticFile(name string) ([]byte, error) {
	if server.staticDir == "" {
		return staticFiles.ReadFile(name)
	}
	rel := strings.TrimPrefix(name, "static/")
	root, err := filepath.Abs(server.staticDir)
	if err != nil {
		return nil, fs.ErrNotExist
	}
	full, err := filepath.Abs(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return nil, fs.ErrNotExist
	}
	// Guard against path traversal outside the configured root.
	if full != root && !strings.HasPrefix(full, root+string(os.PathSeparator)) {
		return nil, fs.ErrNotExist
	}
	return os.ReadFile(full)
}
