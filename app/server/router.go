package server

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

type route struct {
	method       string
	pattern      *regexp.Regexp
	innerHandler http.HandlerFunc
	paramKeys    []string
	isAuth       bool
}

type router struct {
	routes []route
}

func NewRouter() *router {
	return &router{routes: []route{}}
}

func (r *router) GET(isAuth bool, pattern string, handler http.HandlerFunc) {
	r.addRoute(http.MethodGet, isAuth, pattern, handler)
}
func (r *router) POST(isAuth bool, pattern string, handler http.HandlerFunc) {
	r.addRoute(http.MethodPost, isAuth, pattern, handler)
}

func (r *router) addRoute(method string, isAuth bool, endpoint string, handler http.HandlerFunc) {
	// handle path parameters
	pathParamPattern := regexp.MustCompile(":([a-z]+)")
	matches := pathParamPattern.FindAllStringSubmatch(endpoint, -1)
	paramKeys := []string{}
	if len(matches) > 0 {
		// replace path parameter definition with regex pattern to capture any string
		endpoint = pathParamPattern.ReplaceAllLiteralString(endpoint, "([^/]+)")
		// store the names of path parameters, to later be used as context keys
		for i := 0; i < len(matches); i++ {
			paramKeys = append(paramKeys, matches[i][1])
		}
	}

	route := route{method, regexp.MustCompile("^" + endpoint + "$"), handler, paramKeys, isAuth}
	r.routes = append(r.routes, route)
}

// A wrapper around a route's handler, used for logging
func (r *route) handler(w http.ResponseWriter, req *http.Request) {
	requestString := fmt.Sprint(req.Method, " ", req.URL)
	fmt.Println("received ", requestString)
	r.innerHandler(w, req)
}

func (r *router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var allow []string
	for _, route := range r.routes {
		matches := route.pattern.FindStringSubmatch(req.URL.Path)
		if len(matches) > 0 {
			if req.Method != route.method {
				allow = append(allow, route.method)
				continue
			}
			claim, _ := GetAuthCookie(req)
			urlPath := make(map[string]string)
			for i := 0; i < len(route.paramKeys); i++ {
				urlPath[route.paramKeys[i]] = matches[i+1]
			}
			route.handler(w, buildContext(req, urlPath, claim))
			return
		}
	}
	if len(allow) > 0 {
		w.Header().Set("Allow", strings.Join(allow, ", "))
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	//process static files
	if strings.HasPrefix(req.URL.Path, "/js/") {
		http.ServeFile(w, req, "ui/static"+strings.TrimSuffix(req.URL.Path, "/+esm"))
	}
	if strings.HasPrefix(req.URL.Path, "/css/") {
		http.ServeFile(w, req, "ui/static"+req.URL.Path)
	}
	http.NotFound(w, req)
}

// This is used to avoid context key collisions
// it serves as a domain for the context keys
type ContextKey string

// Returns a shallow-copy of the request with an updated context,
// including path parameters
func buildContext(req *http.Request, urlPath map[string]string, claim map[string]string) *http.Request {
	ctx := req.Context()
	if urlPath != nil {
		ctx = context.WithValue(ctx, ContextKey("path"), urlPath)
	}
	if claim != nil {
		ctx = context.WithValue(ctx, ContextKey("claims"), claim)
	}

	return req.WithContext(ctx)
}
