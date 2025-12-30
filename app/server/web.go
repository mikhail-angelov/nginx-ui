package server

import (
	"embed"
	"log"
	"net/http"
	"time"
)

// IsAuth is a constant used to indicate authentication is required for routes.
const IsAuth = true

// Web represents the web server and its routes.
type Web struct {
	router  *Router
	nginx   *Nginx
	service *Service
	email   string
	pass    string
	embedFs embed.FS
}

// NewWeb creates a new Web instance with the given dependencies.
func NewWeb(nginx *Nginx, service *Service, config *Config, embedFs embed.FS) *Web {
	web := &Web{
		router:  NewRouter(embedFs),
		nginx:   nginx,
		service: service,
		email:   config.Email,
		pass:    config.Pass,
		embedFs: embedFs,
	}

	templates := NewTemplate(embedFs)

	web.router.GET(IsAuth, "/test/:id", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("test id - %s", r.Context().Value(ContextKey("id")))

		data := map[string]any{
			"Name": r.Context().Value(ContextKey("id")),
		}

		if err := templates.Render(w, "main", data); err != nil {
			log.Printf("Failed to render main template: %v", err)
		}
	})
	web.router.GET(IsAuth, "/", func(w http.ResponseWriter, r *http.Request) {
		data := make(map[string]any)
		claim := r.Context().Value(ContextKey("claims"))
		errMsg := ""
		if claim == nil {
			data["IsAuth"] = false
		} else {
			configs := service.GetDomains()

			data["IsAuth"] = true
			data["Configs"] = configs
			data["Error"] = errMsg
		}
		if err := templates.Render(w, "index", data); err != nil {
			log.Printf("Failed to render index template: %v", err)
		}

	})
	web.router.GET(IsAuth, "/configs", func(w http.ResponseWriter, r *http.Request) {
		data := make(map[string]any)
		claim := r.Context().Value(ContextKey("claims"))
		errMsg := ""
		if claim == nil {
			data["IsAuth"] = false
		} else {
			configs := service.GetDomains()

			data["IsAuth"] = true
			data["Configs"] = configs
			data["Error"] = errMsg
		}
		if err := templates.SubRender(w, "index", "configs", data); err != nil {
			log.Printf("Failed to render configs template: %v", err)
		}

	})
	web.router.GET(IsAuth, "/edit/{domain}", func(w http.ResponseWriter, r *http.Request) {
		errMsg := ""
		name := r.PathValue("domain")

		content, err := nginx.GetConfig(name)
		if err != nil {
			errMsg = err.Error()
		}
		configs := service.GetDomains()

		data := map[string]any{
			"Configs": configs,
			"Name":    name,
			"Content": content,
			"Error":   errMsg,
		}

		if err := templates.SubRender(w, "index", "editor", data); err != nil {
			log.Printf("Failed to render editor template: %v", err)
		}
	})

	web.router.GET(IsAuth, "/add-config-panel", func(w http.ResponseWriter, _ *http.Request) {
		if err := templates.SubRender(w, "index", "addConfig", nil); err != nil {
			log.Printf("Failed to render addConfig template: %v", err)
		}
	})

	web.router.POST(IsAuth, "/add-config", func(w http.ResponseWriter, r *http.Request) {
		errMsg := ""

		now := time.Now()
		name := r.FormValue("name")
		if name == "" {
			name = now.Format("2024-10-01-15-04-05")
		}

		content, err := service.AddDomain(name)
		if err != nil {
			log.Printf("Failed to add domain %s: %v", name, err)
			errMsg = err.Error()
		}
		configs := service.GetDomains()
		data := map[string]any{
			"Configs": configs,
			"Name":    name,
			"Content": content,
			"Error":   errMsg,
		}
		w.Header().Set("HX-Trigger", "refreshConfigs")
		if err := templates.SubRender(w, "index", "editor", data); err != nil {
			log.Printf("Failed to render editor template: %v", err)
		}
	})
	web.router.POST(IsAuth, "/validate/{domain}", func(w http.ResponseWriter, r *http.Request) {
		errMsg := ""
		name := r.PathValue("domain")
		content := r.FormValue("content")
		err := nginx.CheckNewConfig(name, content)
		status := "valid"
		if err != nil {
			log.Printf("Failed to validate config %s: %v", name, err)
			errMsg = err.Error()
			status = "invalid"
		}

		data := map[string]any{
			"Status": status,
			"Error":  errMsg,
		}

		if err := templates.SubRender(w, "index", "status", data); err != nil {
			log.Printf("Failed to render status template: %v", err)
		}
	})
	web.router.POST(IsAuth, "/save/{domain}", func(w http.ResponseWriter, r *http.Request) {
		errMsg := ""
		name := r.PathValue("domain")
		content := r.FormValue("content")
		err := nginx.SetConfig(name, content)
		status := "valid"
		if err != nil {
			log.Printf("Failed to save config %s: %v", name, err)
			errMsg = err.Error()
			status = "invalid"
		}

		data := map[string]any{
			"Status": status,
			"Error":  errMsg,
		}

		if err := templates.SubRender(w, "index", "status", data); err != nil {
			log.Printf("Failed to render status template: %v", err)
		}
	})
	web.router.POST(IsAuth, "/remove/{domain}", func(w http.ResponseWriter, r *http.Request) {
		errMsg := ""
		name := r.PathValue("domain")
		err := service.RemoveDomain(name)
		if err != nil {
			log.Printf("Failed to remove domain %s: %v", name, err)
			errMsg = err.Error()
		}

		configs := service.GetDomains()

		data := map[string]any{
			"IsAuth":  true,
			"Configs": configs,
			"Error":   errMsg,
		}

		w.Header().Set("HX-Trigger", "refreshConfigs")
		if err := templates.SubRender(w, "index", "dashboard", data); err != nil {
			log.Printf("Failed to render dashboard template: %v", err)
		}
	})

	web.router.POST(false, "/login", func(w http.ResponseWriter, r *http.Request) {
		// validate email and password
		email := r.FormValue("email")
		password := r.FormValue("password")
		data := make(map[string]any)
		log.Printf("Authorizing as %s", email)
		errMsg := ""
		if password != config.Pass || config.Email != email {
			data["IsAuth"] = false
			data["Error"] = "Password is required"
			log.Printf("Login is invalid %s:%s", email, password)
			if err := templates.SubRender(w, "index", "login", data); err != nil {
				log.Printf("Failed to render login template: %v", err)
			}
		} else {
			configs := service.GetDomains()

			data["IsAuth"] = true
			data["Configs"] = configs
			data["Error"] = errMsg
			if err := SetAuthCookie(w, r.FormValue("email")); err != nil {
				log.Printf("Failed to set auth cookie: %v", err)
			}
			if err := templates.SubRender(w, "index", "main", data); err != nil {
				log.Printf("Failed to render main template: %v", err)
			}
		}
	})

	web.router.POST(false, "/logout", func(w http.ResponseWriter, _ *http.Request) {
		CleanAuthCookie(w)
		data := make(map[string]any)
		data["IsAuth"] = false
		if err := templates.SubRender(w, "index", "login", data); err != nil {
			log.Printf("Failed to render login template: %v", err)
		}
	})

	return web
}

// GetRouter returns the underlying http.ServeMux
func (web *Web) GetRouter() *http.ServeMux {
	return web.router.mux
}
