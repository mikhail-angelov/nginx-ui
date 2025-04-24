package server

import (
	"embed"
	"log"
	"net/http"
	"time"
)

const IS_AUTH = true

type Web struct {
	router  *Router
	nginx   *nginx
	service *Service
	email   string
	pass    string
	embedFs embed.FS
}

func NewWeb(nginx *nginx, service *Service, config *Config, embedFs embed.FS) *Web {
	web := &Web{
		router:  NewRouter(embedFs),
		nginx:   nginx,
		service: service,
		email:   config.Email,
		pass:    config.Pass,
		embedFs: embedFs,
	}

	templates := NewTemplate(embedFs)

	web.router.GET(IS_AUTH, "/test/:id", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("test id - %s", r.Context().Value(ContextKey("id")))

		data := map[string]interface{}{
			"Name": r.Context().Value(ContextKey("id")),
		}

		templates.Render(w, "main", data)
	})
	web.router.GET(IS_AUTH, "/", func(w http.ResponseWriter, r *http.Request) {
		data := make(map[string]interface{})
		claim := r.Context().Value(ContextKey("claims"))
		error := ""
		if claim == nil {
			data["IsAuth"] = false
		} else {
			configs := service.GetDomains()

			data["IsAuth"] = true
			data["Configs"] = configs
			data["Error"] = error
		}
		templates.Render(w, "index", data)

	})
	web.router.GET(IS_AUTH, "/configs", func(w http.ResponseWriter, r *http.Request) {
		data := make(map[string]interface{})
		claim := r.Context().Value(ContextKey("claims"))
		error := ""
		if claim == nil {
			data["IsAuth"] = false
		} else {
			configs := service.GetDomains()

			data["IsAuth"] = true
			data["Configs"] = configs
			data["Error"] = error
		}
		templates.SubRender(w, "index", "configs", data)

	})
	web.router.GET(IS_AUTH, "/edit/{domain}", func(w http.ResponseWriter, r *http.Request) {
		error := ""
		name := r.PathValue("domain")

		content, err := nginx.GetConfig(name)
		if err != nil {
			error = err.Error()
		}
		configs := service.GetDomains()

		data := map[string]interface{}{
			"Configs": configs,
			"Name":    name,
			"Content": content,
			"Error":   error,
		}

		templates.SubRender(w, "index", "editor", data)
	})

	web.router.GET(IS_AUTH, "/add-config-panel", func(w http.ResponseWriter, r *http.Request) {
		templates.SubRender(w, "index", "addConfig", nil)
	})

	web.router.POST(IS_AUTH, "/add-config", func(w http.ResponseWriter, r *http.Request) {
		error := ""

		now := time.Now()
		name := r.FormValue("name")
		if name == "" {
			name = now.Format("2024-10-01-15-04-05")
		}

		err, content := service.AddDomain(name)
		if err != nil {
			log.Printf("Failed to add domain %s: %v", name, err)
			error = err.Error()
		}
		configs := service.GetDomains()
		data := map[string]interface{}{
			"Configs": configs,
			"Name":    name,
			"Content": content,
			"Error":   error,
		}
		w.Header().Set("HX-Trigger", "refreshConfigs")
		templates.SubRender(w, "index", "editor", data)
	})
	web.router.POST(IS_AUTH, "/validate/{domain}", func(w http.ResponseWriter, r *http.Request) {
		error := ""
		name := r.PathValue("domain")
		content := r.FormValue("content")
		err := nginx.CheckNewConfig(name, content)
		status := "valid"
		if err != nil {
			log.Printf("Failed to validate config %s: %v", name, err)
			error = err.Error()
			status = "invalid"
		}

		data := map[string]interface{}{
			"Status": status,
			"Error":  error,
		}

		templates.SubRender(w, "index", "status", data)
	})
	web.router.POST(IS_AUTH, "/save/{domain}", func(w http.ResponseWriter, r *http.Request) {
		error := ""
		name := r.PathValue("domain")
		content := r.FormValue("content")
		err := nginx.SetConfig(name, content)
		status := "valid"
		if err != nil {
			log.Printf("Failed to save config %s: %v", name, err)
			error = err.Error()
			status = "invalid"
		}

		data := map[string]interface{}{
			"Status": status,
			"Error":  error,
		}

		templates.SubRender(w, "index", "status", data)
	})
	web.router.POST(IS_AUTH, "/remove/{domain}", func(w http.ResponseWriter, r *http.Request) {
		error := ""
		name := r.PathValue("domain")
		err := service.RemoveDomain(name)
		if err != nil {
			log.Printf("Failed to remove domain %s: %v", name, err)
			error = err.Error()
		}

		configs := service.GetDomains()

		data := map[string]interface{}{
			"IsAuth":  true,
			"Configs": configs,
			"Error":   error,
		}

		w.Header().Set("HX-Trigger", "refreshConfigs")
		templates.SubRender(w, "index", "dashboard", data)
	})

	web.router.POST(false, "/login", func(w http.ResponseWriter, r *http.Request) {
		//validate email and password
		email := r.FormValue("email")
		password := r.FormValue("password")
		data := make(map[string]interface{})
		log.Printf("Authorizing as %s", email)
		error := ""
		if password != config.Pass || config.Email != email {
			data["IsAuth"] = false
			data["Error"] = "Password is required"
			log.Printf("Login is invalid %s:%s", email, password)
			templates.SubRender(w, "index", "login", data)
		} else {
			configs := service.GetDomains()

			data["IsAuth"] = true
			data["Configs"] = configs
			data["Error"] = error
			SetAuthCookie(w, r.FormValue("email"))
			templates.SubRender(w, "index", "main", data)
		}
	})

	web.router.POST(false, "/logout", func(w http.ResponseWriter, r *http.Request) {
		CleanAuthCookie(w)
		data := make(map[string]interface{})
		data["IsAuth"] = false
		templates.SubRender(w, "index", "login", data)
	})

	return web
}

// GetRouter returns the underlying http.ServeMux
func (web *Web) GetRouter() *http.ServeMux {
	return web.router.mux
}
