package server

import (
	"fmt"
	"net/http"
	"time"
)

const IS_AUTH = true

type Web struct {
	router  *router
	nginx   *nginx
	service *Service
}

func NewWeb(nginx *nginx, service *Service) *Web {
	web := &Web{}
	web.nginx = nginx
	web.service = service
	web.router = NewRouter()
	templates := LoadTemplates()

	web.router.GET(IS_AUTH, "/test/:id", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println(r.Context().Value(ContextKey("id"))) // logs the first path parameter

		data := map[string]interface{}{
			"Name": r.Context().Value(ContextKey("id")),
		}

		templates.Render(w, "main", data)
	})
	web.router.GET(IS_AUTH, "/", func(w http.ResponseWriter, r *http.Request) {
		data := make(map[string]interface{})
		claim := r.Context().Value(ContextKey("claims"))
		fmt.Println(claim)
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
		fmt.Println(claim)
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
	web.router.GET(IS_AUTH, "/edit/:file", func(w http.ResponseWriter, r *http.Request) {
		error := ""
		path := r.Context().Value(ContextKey("path")).(map[string]string)
		name := path["file"]

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

	web.router.GET(IS_AUTH, "/add-config-modal", func(w http.ResponseWriter, r *http.Request) {

		templates.SubRender(w, "index", "addConfigModal", nil)
	})
	web.router.POST(IS_AUTH, "/add-config", func(w http.ResponseWriter, r *http.Request) {
		error := ""

		now := time.Now()
		name := r.FormValue("name")
		if name == "" {
			name = now.Format("2024-10-01-15-04-05")
		}
		err := service.AddDomain(name)
		if err != nil {
			error = err.Error()
		}
		configs := service.GetDomains()
		data := map[string]interface{}{
			"Configs": configs,
			"Name":    name,
			"Content": "",
			"Error":   error,
		}
		w.Header().Set("HX-Trigger", "refreshConfigs")
		templates.SubRender(w, "index", "editor", data)
	})
	web.router.POST(IS_AUTH, "/validate/:file", func(w http.ResponseWriter, r *http.Request) {
		error := ""
		path := r.Context().Value(ContextKey("path")).(map[string]string)
		name := path["file"]
		content := r.FormValue("content")
		err := nginx.CheckNewConfig(name, content)
		status := "valid"
		if err != nil {
			error = err.Error()
			status = "invalid"
		}

		data := map[string]interface{}{
			"Status": status,
			"Error":  error,
		}

		templates.SubRender(w, "index", "status", data)
	})
	web.router.POST(IS_AUTH, "/save/:file", func(w http.ResponseWriter, r *http.Request) {
		error := ""
		path := r.Context().Value(ContextKey("path")).(map[string]string)
		name := path["file"]
		content := r.FormValue("content")
		err := nginx.SetConfig(name, content)
		status := "valid"
		if err != nil {
			error = err.Error()
			status = "invalid"
		}

		data := map[string]interface{}{
			"Status": status,
			"Error":  error,
		}

		templates.SubRender(w, "index", "status", data)
	})
	web.router.POST(IS_AUTH, "/remove/:name", func(w http.ResponseWriter, r *http.Request) {
		error := ""
		path := r.Context().Value(ContextKey("path")).(map[string]string)
		name := path["name"]
		err := service.RemoveDomain(name)
		if err != nil {
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
		fmt.Println(email + ":" + password)
		error := ""
		if password == "" {
			data["IsAuth"] = false
			data["Error"] = "Password is required"
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

func (w *Web) GetRouter() *router {
	return w.router
}
