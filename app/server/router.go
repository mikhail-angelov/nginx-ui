package server

import (
	"context"
	"embed"
	"io/fs"
	"net/http"
)

// Router is a simple HTTP router
type Router struct {
	mux     *http.ServeMux
	embedFs embed.FS
}

// NewRouter creates a new Router
func NewRouter(embedFs embed.FS) *Router {
	r := &Router{mux: http.NewServeMux()}

	staticFs, _ := fs.Sub(embedFs, "ui")
	r.mux.Handle("/static/", http.FileServer(http.FS(staticFs)))

	return r
}

// GET registers a new GET route
func (r *Router) GET(auth bool, pattern string, handler http.HandlerFunc) {
	r.mux.HandleFunc(pattern, r.withContext(handler))
}

// POST registers a new POST route
func (r *Router) POST(auth bool, pattern string, handler http.HandlerFunc) {
	r.mux.HandleFunc(pattern, r.withContext(handler))
}

// GetRouter returns the underlying http.ServeMux
func (r *Router) GetRouter() *http.ServeMux {
	return r.mux
}

func (r *Router) withContext(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		next(w, withContext(req))
	}
}

// This is used to avoid context key collisions
// it serves as a domain for the context keys
type ContextKey string

// Returns a shallow-copy of the request with an updated context,
// including path parameters
func withContext(req *http.Request) *http.Request {
	ctx := req.Context()
	claim, _ := GetAuthCookie(req)
	if claim != nil {
		ctx = context.WithValue(ctx, ContextKey("claims"), claim)
	}

	return req.WithContext(ctx)
}
